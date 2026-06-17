package students

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"somotracker/backend/internal/database"
)

// ─── CSV validation ────────────────────────────────────────────────────────

var allowedGenders = map[string]bool{
	"MALE":              true,
	"FEMALE":            true,
	"OTHER":             true,
	"PREFER_NOT_TO_SAY": true,
}

// expectedCSVHeaders defines the canonical column order for student CSV imports.
var expectedCSVHeaders = []string{"first_name", "middle_name", "last_name", "gender", "date_of_birth"}

// ─── Import tracker — in-memory goroutine-safe map ────────────────────────

type importState struct {
	Ch       chan ImportProgress
	Done     chan struct{}
	TenantID string
}

type importTracker struct {
	mu   sync.RWMutex
	jobs map[string]*importState
}

func newImportTracker() *importTracker {
	return &importTracker{
		jobs: make(map[string]*importState),
	}
}

func (t *importTracker) Register(id, tenantID string) *importState {
	state := &importState{
		Ch:       make(chan ImportProgress, 1024),
		Done:     make(chan struct{}),
		TenantID: tenantID,
	}
	t.mu.Lock()
	t.jobs[id] = state
	t.mu.Unlock()
	return state
}

func (t *importTracker) Get(id string) *importState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.jobs[id]
}

func (t *importTracker) Remove(id string) {
	t.mu.Lock()
	delete(t.jobs, id)
	t.mu.Unlock()
}

// ─── Service ──────────────────────────────────────────────────────────────

// Service contains business logic for the students domain.
type Service struct {
	repo          *Repository
	rdb           *redis.Client
	importTracker *importTracker
}

// NewService creates a new Service.
func NewService(repo *Repository, pools *database.Pools) *Service {
	return &Service{
		repo:          repo,
		rdb:           pools.Redis,
		importTracker: newImportTracker(),
	}
}

// ListStudents returns paginated students for a tenant.
func (s *Service) ListStudents(ctx context.Context, tenantID string, offset, limit int, search string) ([]Student, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListByTenant(ctx, tenantID, offset, limit, search)
}

// CreateStudent creates a single student.
func (s *Service) CreateStudent(ctx context.Context, tenantID string, payload CreateStudentPayload) (*Student, error) {
	if payload.FirstName == "" || payload.LastName == "" {
		return nil, fmt.Errorf("first_name and last_name are required")
	}
	if payload.Gender == "" || !allowedGenders[payload.Gender] {
		return nil, fmt.Errorf("gender must be one of: MALE, FEMALE, OTHER, PREFER_NOT_TO_SAY")
	}
	if payload.DateOfBirth == "" {
		return nil, fmt.Errorf("date_of_birth is required")
	}
	// Validate date format
	if _, err := time.Parse("2006-01-02", payload.DateOfBirth); err != nil {
		return nil, fmt.Errorf("date_of_birth must be in YYYY-MM-DD format")
	}

	return s.repo.Create(ctx, tenantID, payload)
}

// ─── CSV Import Pipeline ──────────────────────────────────────────────────

// ImportCSV starts a background ingestion of a CSV blob.
// Returns an import ID immediately. Progress can be tracked via SSE.
func (s *Service) ImportCSV(ctx context.Context, tenantID string, csvData io.Reader) (string, error) {
	// Read all data first (we need it in the goroutine)
	data, err := io.ReadAll(csvData)
	if err != nil {
		return "", fmt.Errorf("read csv data: %w", err)
	}

	importID := generateImportID()

	// Register the import job
	state := s.importTracker.Register(importID, tenantID)

	// Spin up background goroutine — no queue system needed
	go s.processImport(importID, tenantID, data, state)

	return importID, nil
}

// GetImportStream returns the progress channel for an import job.
func (s *Service) GetImportStream(importID string) <-chan ImportProgress {
	state := s.importTracker.Get(importID)
	if state == nil {
		return nil
	}
	return state.Ch
}

// GetImportDone returns the done channel for an import job.
func (s *Service) GetImportDone(importID string) <-chan struct{} {
	state := s.importTracker.Get(importID)
	if state == nil {
		return nil
	}
	return state.Done
}

// GetErrorCSV retrieves the error CSV from Redis.
func (s *Service) GetErrorCSV(ctx context.Context, errorID string) (string, error) {
	data, err := s.rdb.Get(ctx, "err_log:"+errorID).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("error log not found or expired")
	}
	if err != nil {
		return "", fmt.Errorf("redis get: %w", err)
	}
	return data, nil
}

// ─── Internal processing ──────────────────────────────────────────────────

func (s *Service) processImport(importID, tenantID string, data []byte, state *importState) {
	defer func() {
		close(state.Done)
		// Cleanup tracker after a delay (allow SSE consumers to read final state)
		time.AfterFunc(30*time.Second, func() { s.importTracker.Remove(importID) })
	}()

	// Parse CSV
	reader := csv.NewReader(bytes.NewReader(data))
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = false

	allRecords, err := reader.ReadAll()
	if err != nil {
		state.Ch <- ImportProgress{Status: "error", Error: "failed to parse CSV: " + err.Error()}
		return
	}

	if len(allRecords) < 2 {
		state.Ch <- ImportProgress{Status: "error", Error: "CSV must have a header row and at least one data row"}
		return
	}

	// Validate header
	header := allRecords[0]
	normalizedHeader := normalizeHeaders(header)
	if !validateHeaders(normalizedHeader) {
		var missing []string
		for _, expected := range expectedCSVHeaders {
			found := false
			for _, h := range normalizedHeader {
				if h == expected {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, expected)
			}
		}
		state.Ch <- ImportProgress{
			Status: "error",
			Error:  fmt.Sprintf("CSV missing required columns: %s. Expected: first_name, middle_name, last_name, gender, date_of_birth", strings.Join(missing, ", ")),
		}
		return
	}

	// Build header index map
	headerIdx := make(map[string]int)
	for i, h := range normalizedHeader {
		headerIdx[h] = i
	}

	totalRows := len(allRecords) - 1
	var validRows []CSVRawRow
	var errorRows []CSVRawRow
	var errorReasons []string

	for i, record := range allRecords[1:] {
		lineNum := i + 2 // 1-indexed + header
		row := CSVRawRow{
			FirstName:   getField(record, headerIdx, "first_name"),
			MiddleName:  getField(record, headerIdx, "middle_name"),
			LastName:    getField(record, headerIdx, "last_name"),
			Gender:      strings.ToUpper(getField(record, headerIdx, "gender")),
			DateOfBirth: getField(record, headerIdx, "date_of_birth"),
			LineNumber:  lineNum,
		}

		// Validate row
		if reason := validateRow(row); reason != "" {
			errorRows = append(errorRows, row)
			errorReasons = append(errorReasons, reason)
		} else {
			validRows = append(validRows, row)
		}

		// Send progress every 50 rows or on last row
		if (i+1)%50 == 0 || i+1 == totalRows {
			state.Ch <- ImportProgress{
				Status:  "processing",
				Current: i + 1,
				Total:   totalRows,
			}
		}
	}

	// Bulk insert valid rows
	insertedCount := 0
	if len(validRows) > 0 {
		inserted, err := s.repo.BulkInsert(context.Background(), tenantID, validRows)
		if err != nil {
			state.Ch <- ImportProgress{Status: "error", Error: "database insert failed: " + err.Error()}
			return
		}
		insertedCount = inserted
	}

	// Store error CSV in Redis if there are failures
	var downloadURL string
	if len(errorRows) > 0 {
		errCSV := buildErrorCSV(errorRows, errorReasons)
		errorID := importID
		if err := s.rdb.Set(context.Background(), "err_log:"+errorID, errCSV, 1800*time.Second).Err(); err != nil {
			// Non-fatal: log and proceed without error download
			state.Ch <- ImportProgress{
				Status:  "completed",
				Success: insertedCount,
				Failed:  len(errorRows),
				Total:   totalRows,
			}
			return
		}
		downloadURL = fmt.Sprintf("/api/v1/students/import/errors?id=%s", errorID)
	}

	// Final progress — terminal event
	state.Ch <- ImportProgress{
		Status:  "completed",
		Success: insertedCount,
		Failed:  len(errorRows),
		Total:   totalRows,
		Error:   downloadURL,
	}
}

// ─── Validation helpers ───────────────────────────────────────────────────

func validateRow(row CSVRawRow) string {
	if row.FirstName == "" {
		return "first_name is required"
	}
	if row.LastName == "" {
		return "last_name is required"
	}
	if !allowedGenders[row.Gender] {
		return fmt.Sprintf("invalid gender '%s'; must be one of: MALE, FEMALE, OTHER, PREFER_NOT_TO_SAY", row.Gender)
	}
	if _, err := time.Parse("2006-01-02", row.DateOfBirth); err != nil {
		return fmt.Sprintf("invalid date_of_birth '%s'; must be YYYY-MM-DD", row.DateOfBirth)
	}
	return ""
}

func normalizeHeaders(headers []string) []string {
	normalized := make([]string, len(headers))
	for i, h := range headers {
		h = strings.TrimSpace(strings.ToLower(h))
		h = strings.ReplaceAll(h, " ", "_")
		h = strings.ReplaceAll(h, "-", "_")
		normalized[i] = h
	}
	return normalized
}

func validateHeaders(headers []string) bool {
	headerSet := make(map[string]bool, len(headers))
	for _, h := range headers {
		headerSet[h] = true
	}
	for _, expected := range expectedCSVHeaders {
		if !headerSet[expected] {
			return false
		}
	}
	return true
}

func getField(record []string, idx map[string]int, field string) string {
	if i, ok := idx[field]; ok && i < len(record) {
		return strings.TrimSpace(record[i])
	}
	return ""
}

// ─── Error CSV builder ────────────────────────────────────────────────────

func buildErrorCSV(rows []CSVRawRow, reasons []string) string {
	var buf strings.Builder
	buf.WriteString("first_name,middle_name,last_name,gender,date_of_birth,reason_for_failure\n")
	for i, row := range rows {
		reason := ""
		if i < len(reasons) {
			reason = escapeCSVField(reasons[i])
		}
		_, _ = fmt.Fprintf(&buf, "%s,%s,%s,%s,%s,%s\n",
			escapeCSVField(row.FirstName),
			escapeCSVField(row.MiddleName),
			escapeCSVField(row.LastName),
			row.Gender,
			row.DateOfBirth,
			reason,
		)
	}
	return buf.String()
}

func escapeCSVField(s string) string {
	if strings.ContainsAny(s, `,"\n`) {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// ─── ID generation ────────────────────────────────────────────────────────

var idMu sync.Mutex
var idCounter int64

func generateImportID() string {
	idMu.Lock()
	idCounter++
	now := time.Now().UnixNano()
	id := fmt.Sprintf("import_%d_%d", now, idCounter)
	idMu.Unlock()
	return id
}
