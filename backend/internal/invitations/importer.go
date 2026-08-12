package invitations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"

	"somotracker/backend/internal/imports"
)

// ============================================================================
// Constants
// ============================================================================

// invitationExpiryDays is the number of days until a pending invitation expires.
const invitationExpiryDays = 7

// emailRegex is a basic email format validator. It checks for the presence of
// an @ sign with a non-empty local part and domain.
var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// StaffInviteRedirectPath is the backend path that handles invite magic link
// callbacks. Stytch redirects the user to this backend endpoint after they
// click the magic link in the invite email. The endpoint authenticates the
// token, creates the user session, and redirects to the frontend dashboard.
// This URL must be registered as an invite redirect URL in the Stytch dashboard.
const StaffInviteRedirectPath = "/api/auth/invite/callback"

// ============================================================================
// AugmentedRow is what ValidatedRow.RawData becomes after ResolveReferences.
// ============================================================================

// augmentedInviteRow extends InviteRow with the tenant/school/role context
// injected by ResolveReferences plus the staging row ID for tracking.
type augmentedInviteRow struct {
	Email        string `json:"email"`
	FullName     string `json:"full_name,omitempty"`
	TenantID     string `json:"tenant_id"`
	SchoolID     string `json:"school_id"`
	Role         string `json:"role"`
	StytchOrgID  string `json:"stytch_org_id"`
	InvitedBy    string `json:"invited_by"`
	ImportJobID  string `json:"import_job_id"`
	StagingRowID string `json:"staging_row_id,omitempty"`
}

// ============================================================================
// StaffInviteImporter — implements imports.Importer for STAFF_INVITE jobs
// ============================================================================

// StaffInviteImporter handles staff invitation bulk operations.
// It validates emails, checks for duplicates, creates Stytch members,
// sends invite emails, and persists invitation records.
type StaffInviteImporter struct {
	repo       Repository
	stytch     StytchInviteSender
	backendURL string
	logger     *zap.SugaredLogger
}

// NewStaffInviteImporter creates a new StaffInviteImporter.
func NewStaffInviteImporter(repo Repository, logger *zap.SugaredLogger) *StaffInviteImporter {
	return &StaffInviteImporter{
		repo:   repo,
		logger: logger,
	}
}

// SetStytchAdapter sets the Stytch identity provider adapter.
// Called during DI wiring.
func (si *StaffInviteImporter) SetStytchAdapter(stytch StytchInviteSender) {
	si.stytch = stytch
}

// SetBackendURL sets the backend base URL.
// The discovery redirect URL must be registered in the Stytch dashboard.
func (si *StaffInviteImporter) SetBackendURL(url string) {
	si.backendURL = url
}

// JobType returns STAFF_INVITE.
func (si *StaffInviteImporter) JobType() imports.ImportJobType {
	return imports.ImportJobTypeStaffInvite
}

// Validate checks each raw row for schema-level correctness.
// Checks performed per row:
//   - JSON is valid
//   - email is non-empty after trimming
//   - email matches basic format
func (si *StaffInviteImporter) Validate(ctx context.Context, tenantID, schoolID uuid.UUID, raw []json.RawMessage) ([]imports.ValidatedRow, []imports.RowFailure) {
	var validated []imports.ValidatedRow
	var failures []imports.RowFailure

	for i, rawData := range raw {
		var row InviteRow
		if err := json.Unmarshal(rawData, &row); err != nil {
			failures = append(failures, imports.RowFailure{
				RowNumber:    int64(i),
				RawPayload:   rawData,
				ErrorMessage: fmt.Sprintf("invalid JSON: %v", err),
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		// Trim whitespace from email
		row.Email = strings.TrimSpace(row.Email)

		// Required: email must be non-empty
		if row.Email == "" {
			failures = append(failures, imports.RowFailure{
				RowNumber:    int64(i),
				RawPayload:   rawData,
				ErrorMessage: "email is required",
				ErrorType:    imports.ImportFailureInvalidEmailFormat,
			})
			continue
		}

		// Format: email must match basic pattern
		if !emailRegex.MatchString(row.Email) {
			failures = append(failures, imports.RowFailure{
				RowNumber:    int64(i),
				RawPayload:   rawData,
				ErrorMessage: fmt.Sprintf("email %q is not a valid email address", row.Email),
				ErrorType:    imports.ImportFailureInvalidEmailFormat,
			})
			continue
		}

		// Normalize name: trim and limit to empty string
		if row.FullName != nil {
			trimmed := strings.TrimSpace(*row.FullName)
			row.FullName = &trimmed
		}

		// Re-marshal cleaned row so staging data is clean
		cleanData, err := json.Marshal(row)
		if err != nil {
			failures = append(failures, imports.RowFailure{
				RowNumber:    int64(i),
				RawPayload:   rawData,
				ErrorMessage: fmt.Sprintf("marshal cleaned row: %v", err),
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		validated = append(validated, imports.ValidatedRow{RawData: cleanData})
	}

	if len(failures) > 0 {
		si.logger.Debugw("invitations.StaffInviteImporter.Validate: schema validation failures",
			"total", len(raw),
			"valid", len(validated),
			"failed", len(failures),
		)
	}

	return validated, failures
}

// ResolveReferences checks each email against existing users and pending
// invitations, then injects tenant/school/role context into each row.
//
// Two-phase approach:
//
//	Phase 1: Unmarshal all rows and collect all emails
//	Phase 2: Single batch query for duplicate detection
//	Phase 3: Build augmented rows, rejecting duplicates
func (si *StaffInviteImporter) ResolveReferences(ctx context.Context, tenantID, schoolID uuid.UUID, metadata json.RawMessage, rows []imports.ValidatedRow) ([]imports.ValidatedRow, []imports.RowFailure) {
	if len(rows) == 0 {
		return rows, nil
	}

	// Parse job metadata for role, invited_by, import_job_id, stytch_org_id
	var meta struct {
		Role        string `json:"role"`
		InvitedBy   string `json:"invited_by"`
		ImportJobID string `json:"import_job_id"`
		StytchOrgID string `json:"stytch_org_id"`
	}
	if err := json.Unmarshal(metadata, &meta); err != nil {
		si.logger.Errorw("invitations.StaffInviteImporter.ResolveReferences: invalid metadata", "error", err)
		return nil, allInviteFail(rows, "job metadata is invalid")
	}

	if meta.Role == "" {
		return nil, allInviteFail(rows, "job metadata missing role")
	}
	if meta.StytchOrgID == "" {
		return nil, allInviteFail(rows, "tenant has no authentication provider configured")
	}

	// ── Phase 1: Unmarshal all rows and collect emails ──
	type parsedRow struct {
		index     int
		row       InviteRow
		rawData   json.RawMessage
		stagingID uuid.UUID
	}
	parsed := make([]parsedRow, 0, len(rows))
	emailSet := make(map[string]struct{}) // for dedup within the batch
	var allEmails []string

	for i, row := range rows {
		var inviteRow InviteRow
		if err := json.Unmarshal(row.RawData, &inviteRow); err != nil {
			// Should not happen after Validate, but handle defensively
			return nil, []imports.RowFailure{{
				RowNumber:    int64(i),
				RawPayload:   row.RawData,
				ErrorMessage: fmt.Sprintf("unmarshal row: %v", err),
				ErrorType:    imports.ImportFailureSchemaValidation,
			}}
		}

		parsed = append(parsed, parsedRow{
			index:     i,
			row:       inviteRow,
			rawData:   row.RawData,
			stagingID: row.StagingRowID,
		})

		// Track unique emails for batch query
		email := inviteRow.Email
		if _, exists := emailSet[email]; !exists {
			emailSet[email] = struct{}{}
			allEmails = append(allEmails, email)
		}
	}

	// ── Phase 2: Batch query for duplicate detection ──
	var existingUsers, existingInvites []string
	var queryErr error
	if len(allEmails) > 0 {
		existingUsers, existingInvites, queryErr = si.repo.CheckExistingEmails(ctx, tenantID.String(), schoolID.String(), allEmails)
		if queryErr != nil {
			si.logger.Errorw("invitations.StaffInviteImporter.ResolveReferences: batch query failed",
				"error", queryErr,
			)
			return nil, allInviteFail(rows, "could not verify email uniqueness")
		}
	}

	existingUserSet := sliceToSet(existingUsers)
	existingInviteSet := sliceToSet(existingInvites)

	// ── Phase 3: Build augmented rows, rejecting duplicates ──
	var resolved []imports.ValidatedRow
	var failures []imports.RowFailure

	for _, p := range parsed {
		email := p.row.Email

		// Check against existing users (someone already has an account)
		if _, exists := existingUserSet[email]; exists {
			failures = append(failures, imports.RowFailure{
				RowNumber:    int64(p.index),
				RawPayload:   p.rawData,
				ErrorMessage: fmt.Sprintf("Email %s already exists for this school", email),
				ErrorType:    imports.ImportFailureDuplicateEmail,
			})
			continue
		}

		// Check against pending invitations
		if _, exists := existingInviteSet[email]; exists {
			failures = append(failures, imports.RowFailure{
				RowNumber:    int64(p.index),
				RawPayload:   p.rawData,
				ErrorMessage: fmt.Sprintf("Email %s already has a pending invitation for this school", email),
				ErrorType:    imports.ImportFailureDuplicateEmail,
			})
			continue
		}

		// Build augmented row
		name := ""
		if p.row.FullName != nil {
			name = *p.row.FullName
		}

		aug := augmentedInviteRow{
			Email:        email,
			FullName:     name,
			TenantID:     tenantID.String(),
			SchoolID:     schoolID.String(),
			Role:         meta.Role,
			StytchOrgID:  meta.StytchOrgID,
			InvitedBy:    meta.InvitedBy,
			ImportJobID:  meta.ImportJobID,
			StagingRowID: p.stagingID.String(),
		}

		augData, err := json.Marshal(aug)
		if err != nil {
			failures = append(failures, imports.RowFailure{
				RowNumber:    int64(p.index),
				RawPayload:   p.rawData,
				ErrorMessage: fmt.Sprintf("marshal augmented row: %v", err),
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		resolved = append(resolved, imports.ValidatedRow{
			RawData:      augData,
			StagingRowID: p.stagingID,
		})
	}

	return resolved, failures
}

// BulkInsert returns an error to force per-row savepoint fallback.
// Staff invitations require per-row Stytch API calls, so bulk insert
// is not applicable.
func (si *StaffInviteImporter) BulkInsert(ctx context.Context, tx pgx.Tx, rows []imports.ValidatedRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	return 0, fmt.Errorf("staff invite requires per-row Stytch API calls")
}

// InsertOne processes a single invitation row:
//  1. Creates a Stytch member and sends an invitation email via InviteMemberByEmail
//  2. Inserts an invitation record in the DB (within the savepoint) with the
//     Stytch member ID returned from the invite call
//
// If any step fails, the savepoint is rolled back and the staging row stays
// 'pending' for retry via Asynq redelivery.
func (si *StaffInviteImporter) InsertOne(ctx context.Context, tx pgx.Tx, row imports.ValidatedRow) error {
	var aug augmentedInviteRow
	if err := json.Unmarshal(row.RawData, &aug); err != nil {
		return &imports.ImportError{
			Type:    imports.ImportFailureSchemaValidation,
			Message: fmt.Sprintf("unmarshal augmented row: %v", err),
		}
	}

	if si.stytch == nil {
		return &imports.ImportError{
			Type:    imports.ImportFailureStytchAPIError,
			Message: "authentication provider not configured",
		}
	}

	// Step 1: Create Stytch member and send invitation email.
	// InviteMemberByEmail both creates the member in Stytch and dispatches
	// a proper invitation email (not a login/discovery email).
	inviteCallbackURL := si.backendURL + StaffInviteRedirectPath
	stytchMemberID, err := si.stytch.InviteMemberByEmail(ctx, aug.StytchOrgID, aug.Email, aug.FullName, inviteCallbackURL)
	if err != nil {
		return &imports.ImportError{
			Type:    imports.ImportFailureStytchAPIError,
			Message: fmt.Sprintf("Could not send invitation email: %v", err),
		}
	}

	// Step 2: Insert invitation record in DB (within savepoint) with the
	// Stytch member ID from the invite call.
	// Parse all UUID-valued columns from JSON strings to uuid.UUID.
	// uuid.Parse of an invalid/empty string returns uuid.Nil, which is handled
	// by uuidToPtr in the repository (nil → SQL NULL for nullable columns).
	params := InsertInvitationParams{
		Email:          aug.Email,
		FullName:       aug.FullName,
		TenantID:       uuid.MustParse(aug.TenantID),
		SchoolID:       uuid.MustParse(aug.SchoolID),
		Role:           aug.Role,
		InvitedBy:      mustParseUUID(aug.InvitedBy),
		Status:         "pending",
		StytchMemberID: stytchMemberID,
		ExpiresAt:      time.Now().Add(invitationExpiryDays * 24 * time.Hour),
		ImportJobID:    mustParseUUID(aug.ImportJobID),
	}

	if err := si.repo.InsertInvitation(ctx, tx, params); err != nil {
		// Check for unique constraint violation on uq_invitations_school_email_pending
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return &imports.ImportError{
				Type:    imports.ImportFailureDuplicateEmail,
				Message: fmt.Sprintf("Email %s already has a pending invitation for this school", aug.Email),
			}
		}
		return &imports.ImportError{
			Type:    imports.ImportFailureInviteInsertFailed,
			Message: fmt.Sprintf("Could not save invitation record: %v", err),
		}
	}

	return nil
}

// ============================================================================
// Helpers
// ============================================================================

// allInviteFail creates a failure for every row with the given message.
func allInviteFail(rows []imports.ValidatedRow, msg string) []imports.RowFailure {
	failures := make([]imports.RowFailure, 0, len(rows))
	for i, row := range rows {
		failures = append(failures, imports.RowFailure{
			RowNumber:    int64(i),
			RawPayload:   row.RawData,
			ErrorMessage: msg,
			ErrorType:    imports.ImportFailureBusinessRule,
		})
	}
	return failures
}

// mustParseUUID parses a UUID string, returning uuid.Nil for empty or invalid
// inputs. This is used at JSON boundaries where UUID fields arrive as strings
// and nullable columns should map to uuid.Nil (→ SQL NULL via uuidToPtr).
func mustParseUUID(s string) uuid.UUID {
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return u
}

// sliceToSet converts a string slice to a set for O(1) lookups.
func sliceToSet(slice []string) map[string]struct{} {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	return set
}

// compile-time interface check
var _ imports.Importer = (*StaffInviteImporter)(nil)
