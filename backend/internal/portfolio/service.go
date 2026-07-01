package portfolio

import (
	"context"
	"fmt"
	"log/slog"
)

// ============================================================================
// Service
// ============================================================================

// Service contains business logic for portfolio entries.
type Service struct {
	Repo              Repository
	StudentResolver   StudentResolver
	SubStrandResolver SubStrandResolver
}

// NewService creates a new Service.
func NewService(repo Repository, studentResolver StudentResolver, subStrandResolver SubStrandResolver) *Service {
	return &Service{
		Repo:              repo,
		StudentResolver:   studentResolver,
		SubStrandResolver: subStrandResolver,
	}
}

// ============================================================================
// CREATE
// ============================================================================

// CreateEntry validates and creates a new portfolio entry.
func (s *Service) CreateEntry(ctx context.Context, tenantID string, payload CreatePortfolioEntryPayload) (*PortfolioEntry, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("portfolio.Service.CreateEntry: %w", ErrInvalidInput)
	}

	// Validate required fields
	if payload.StudentID == "" {
		return nil, fmt.Errorf("portfolio.Service.CreateEntry: student_id is required: %w", ErrInvalidInput)
	}
	if payload.SubStrandID == "" {
		return nil, fmt.Errorf("portfolio.Service.CreateEntry: sub_strand_id is required: %w", ErrInvalidInput)
	}
	if payload.EvidenceType == "" {
		return nil, fmt.Errorf("portfolio.Service.CreateEntry: evidence_type is required: %w", ErrInvalidInput)
	}
	if payload.StoragePointer == "" {
		return nil, fmt.Errorf("portfolio.Service.CreateEntry: storage_pointer is required: %w", ErrInvalidInput)
	}

	// Validate evidence_type
	if !validEvidenceTypes[payload.EvidenceType] {
		return nil, fmt.Errorf("portfolio.Service.CreateEntry: evidence_type %q: %w", payload.EvidenceType, ErrInvalidEvidenceType)
	}

	// Validate date_collected format if provided
	if payload.DateCollected != nil && *payload.DateCollected != "" {
		if !isValidDate(*payload.DateCollected) {
			return nil, fmt.Errorf("portfolio.Service.CreateEntry: date_collected must be YYYY-MM-DD: %w", ErrInvalidInput)
		}
	}

	// Validate student exists
	studentExists, err := s.StudentResolver.StudentExists(ctx, tenantID, payload.StudentID)
	if err != nil {
		return nil, fmt.Errorf("portfolio.Service.CreateEntry: %w", err)
	}
	if !studentExists {
		return nil, fmt.Errorf("portfolio.Service.CreateEntry: student %s: %w", payload.StudentID, ErrStudentNotFound)
	}

	// Validate sub_strand exists
	subStrandExists, err := s.SubStrandResolver.SubStrandExists(ctx, payload.SubStrandID)
	if err != nil {
		return nil, fmt.Errorf("portfolio.Service.CreateEntry: %w", err)
	}
	if !subStrandExists {
		return nil, fmt.Errorf("portfolio.Service.CreateEntry: sub_strand %s: %w", payload.SubStrandID, ErrSubStrandNotFound)
	}

	// Advisory duplicate check — advise against duplicates but don't block
	exists, err := s.Repo.EntryExistsForStudentSubStrandEvidence(ctx, tenantID, payload.StudentID, payload.SubStrandID, payload.EvidenceType)
	if err != nil {
		return nil, fmt.Errorf("portfolio.Service.CreateEntry: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("portfolio.Service.CreateEntry: %w", ErrDuplicateAdvised)
	}

	entry := &PortfolioEntry{
		TenantID:       tenantID,
		StudentID:      payload.StudentID,
		SubStrandID:    payload.SubStrandID,
		EvidenceType:   payload.EvidenceType,
		StoragePointer: payload.StoragePointer,
		LinkedResultID: payload.LinkedResultID,
		DateCollected:  payload.DateCollected,
	}

	id, err := s.Repo.CreateEntry(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("portfolio.Service.CreateEntry: %w", err)
	}
	entry.ID = id

	slog.Info("portfolio_entry.created",
		"tenant_id", tenantID,
		"resource_id", id,
		"student_id", payload.StudentID,
		"sub_strand_id", payload.SubStrandID,
		"evidence_type", payload.EvidenceType,
	)

	return entry, nil
}

// ============================================================================
// LIST
// ============================================================================

// ListEntries returns portfolio entries matching the given filters.
func (s *Service) ListEntries(ctx context.Context, tenantID string, query ListEntriesQuery) ([]PortfolioEntry, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("portfolio.Service.ListEntries: %w", ErrInvalidInput)
	}
	return s.Repo.ListEntries(ctx, tenantID, query)
}

// ============================================================================
// GET
// ============================================================================

// GetEntry returns a single portfolio entry by ID.
func (s *Service) GetEntry(ctx context.Context, id, tenantID string) (*PortfolioEntry, error) {
	if id == "" || tenantID == "" {
		return nil, fmt.Errorf("portfolio.Service.GetEntry: %w", ErrInvalidInput)
	}
	return s.Repo.GetEntryByID(ctx, id, tenantID)
}

// ============================================================================
// UPDATE
// ============================================================================

// UpdateEntry applies partial updates to a portfolio entry.
func (s *Service) UpdateEntry(ctx context.Context, id, tenantID string, payload UpdatePortfolioEntryPayload) error {
	if id == "" || tenantID == "" {
		return fmt.Errorf("portfolio.Service.UpdateEntry: %w", ErrInvalidInput)
	}

	entry, err := s.Repo.GetEntryByID(ctx, id, tenantID)
	if err != nil {
		return fmt.Errorf("portfolio.Service.UpdateEntry: %w", err)
	}

	// Apply partial updates
	if payload.StoragePointer != nil {
		if *payload.StoragePointer == "" {
			return fmt.Errorf("portfolio.Service.UpdateEntry: storage_pointer cannot be empty: %w", ErrInvalidInput)
		}
		entry.StoragePointer = *payload.StoragePointer
	}

	if payload.DateCollected != nil {
		if *payload.DateCollected == "" {
			// Explicitly set to empty string — we treat as clearing the date
			entry.DateCollected = nil
		} else {
			if !isValidDate(*payload.DateCollected) {
				return fmt.Errorf("portfolio.Service.UpdateEntry: date_collected must be YYYY-MM-DD: %w", ErrInvalidInput)
			}
			entry.DateCollected = payload.DateCollected
		}
	}

	if payload.LinkedResultID != nil {
		if *payload.LinkedResultID == "" {
			// Empty string means unlink
			entry.LinkedResultID = nil
		} else {
			entry.LinkedResultID = payload.LinkedResultID
		}
	}

	if err := s.Repo.UpdateEntry(ctx, entry); err != nil {
		return fmt.Errorf("portfolio.Service.UpdateEntry: %w", err)
	}

	slog.Info("portfolio_entry.updated",
		"tenant_id", tenantID,
		"resource_id", id,
		"storage_pointer_changed", payload.StoragePointer != nil,
		"date_collected_changed", payload.DateCollected != nil,
		"linked_result_changed", payload.LinkedResultID != nil,
	)

	return nil
}

// ============================================================================
// DELETE
// ============================================================================

// DeleteEntry removes a portfolio entry.
func (s *Service) DeleteEntry(ctx context.Context, id, tenantID string) error {
	if id == "" || tenantID == "" {
		return fmt.Errorf("portfolio.Service.DeleteEntry: %w", ErrInvalidInput)
	}
	return s.Repo.DeleteEntry(ctx, id, tenantID)
}

// ============================================================================
// Helpers
// ============================================================================

// isValidDate checks if the given string is a valid YYYY-MM-DD date.
func isValidDate(s string) bool {
	if len(s) != 10 {
		return false
	}
	if s[4] != '-' || s[7] != '-' {
		return false
	}
	// Basic range checks
	year := s[0:4]
	month := s[5:7]
	day := s[8:10]
	for _, c := range year {
		if c < '0' || c > '9' {
			return false
		}
	}
	for _, c := range month {
		if c < '0' || c > '9' {
			return false
		}
	}
	for _, c := range day {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
