package invitations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"somotracker/backend/internal/imports"
)

// ============================================================================
// Constants
// ============================================================================

// parentInviteExpiryDays is the number of days until a pending parent invitation expires.
const parentInviteExpiryDays = 7

// parentEmailRegex is a basic email format validator.
var parentEmailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// ParentInviteRedirectPath is the backend path that handles parent invite magic link callbacks.
const ParentInviteRedirectPath = "/api/auth/invite/callback"

// ============================================================================
// AugmentedRow for parent invitations
// ============================================================================

// augmentedParentInviteRow extends InviteRow with tenant/school/role context.
type augmentedParentInviteRow struct {
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
// ParentInviteImporter — implements imports.Importer for PARENT_INVITE jobs
// ============================================================================

// ParentInviteImporter handles parent invitation bulk operations.
// It validates emails, checks for duplicates, creates Stytch members,
// sends invite emails, and persists invitation records with role=PARENT.
type ParentInviteImporter struct {
	repo       Repository
	stytch     StytchInviteSender
	backendURL string
}

// NewParentInviteImporter creates a new ParentInviteImporter.
func NewParentInviteImporter(repo Repository) *ParentInviteImporter {
	return &ParentInviteImporter{
		repo: repo,
	}
}

// SetStytchAdapter sets the Stytch identity provider adapter.
// Called during DI wiring.
func (pi *ParentInviteImporter) SetStytchAdapter(stytch StytchInviteSender) {
	pi.stytch = stytch
}

// SetBackendURL sets the backend base URL.
func (pi *ParentInviteImporter) SetBackendURL(url string) {
	pi.backendURL = url
}

// JobType returns PARENT_INVITE.
func (pi *ParentInviteImporter) JobType() imports.ImportJobType {
	return imports.ImportJobTypeParentInvite
}

// Validate checks each raw row for schema-level correctness.
func (pi *ParentInviteImporter) Validate(ctx context.Context, tenantID, schoolID uuid.UUID, raw []json.RawMessage) ([]imports.ValidatedRow, []imports.RowFailure) {
	var validated []imports.ValidatedRow
	var failures []imports.RowFailure

	for i, rawData := range raw {
		var row InviteRow
		if err := json.Unmarshal(rawData, &row); err != nil {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   rawData,
				ErrorMessage: fmt.Sprintf("invalid JSON: %v", err),
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		row.Email = strings.TrimSpace(row.Email)

		if row.Email == "" {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   rawData,
				ErrorMessage: "email is required",
				ErrorType:    imports.ImportFailureInvalidEmailFormat,
			})
			continue
		}

		if !parentEmailRegex.MatchString(row.Email) {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   rawData,
				ErrorMessage: fmt.Sprintf("email %q is not a valid email address", row.Email),
				ErrorType:    imports.ImportFailureInvalidEmailFormat,
			})
			continue
		}

		if row.FullName != nil {
			trimmed := strings.TrimSpace(*row.FullName)
			row.FullName = &trimmed
		}

		cleanData, err := json.Marshal(row)
		if err != nil {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   rawData,
				ErrorMessage: fmt.Sprintf("marshal cleaned row: %v", err),
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		validated = append(validated, imports.ValidatedRow{RawData: cleanData})
	}

	if len(failures) > 0 {
		slog.Debug("invitations.ParentInviteImporter.Validate: schema validation failures",
			"total", len(raw),
			"valid", len(validated),
			"failed", len(failures),
		)
	}

	return validated, failures
}

// ResolveReferences checks each email against existing users and pending
// invitations, then injects tenant/school/role context into each row.
func (pi *ParentInviteImporter) ResolveReferences(ctx context.Context, tenantID, schoolID uuid.UUID, metadata json.RawMessage, rows []imports.ValidatedRow) ([]imports.ValidatedRow, []imports.RowFailure) {
	if len(rows) == 0 {
		return rows, nil
	}

	var meta struct {
		Role        string `json:"role"`
		InvitedBy   string `json:"invited_by"`
		ImportJobID string `json:"import_job_id"`
		StytchOrgID string `json:"stytch_org_id"`
	}
	if err := json.Unmarshal(metadata, &meta); err != nil {
		slog.Error("invitations.ParentInviteImporter.ResolveReferences: invalid metadata", "error", err)
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
	emailSet := make(map[string]struct{})
	var allEmails []string

	for i, row := range rows {
		var inviteRow InviteRow
		if err := json.Unmarshal(row.RawData, &inviteRow); err != nil {
			return nil, []imports.RowFailure{{
				RowNumber:    i,
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
		existingUsers, existingInvites, queryErr = pi.repo.CheckExistingEmails(ctx, tenantID.String(), schoolID.String(), allEmails)
		if queryErr != nil {
			slog.Error("invitations.ParentInviteImporter.ResolveReferences: batch query failed",
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

		if _, exists := existingUserSet[email]; exists {
			failures = append(failures, imports.RowFailure{
				RowNumber:    p.index,
				RawPayload:   p.rawData,
				ErrorMessage: fmt.Sprintf("Email %s already exists for this school", email),
				ErrorType:    imports.ImportFailureDuplicateEmail,
			})
			continue
		}

		if _, exists := existingInviteSet[email]; exists {
			failures = append(failures, imports.RowFailure{
				RowNumber:    p.index,
				RawPayload:   p.rawData,
				ErrorMessage: fmt.Sprintf("Email %s already has a pending invitation for this school", email),
				ErrorType:    imports.ImportFailureDuplicateEmail,
			})
			continue
		}

		name := ""
		if p.row.FullName != nil {
			name = *p.row.FullName
		}

		aug := augmentedParentInviteRow{
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
				RowNumber:    p.index,
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
func (pi *ParentInviteImporter) BulkInsert(ctx context.Context, tx pgx.Tx, rows []imports.ValidatedRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	return 0, fmt.Errorf("parent invite requires per-row Stytch API calls")
}

// InsertOne processes a single parent invitation row:
//  1. Creates a Stytch member and sends an invitation email
//  2. Inserts an invitation record in the DB with role=PARENT
func (pi *ParentInviteImporter) InsertOne(ctx context.Context, tx pgx.Tx, row imports.ValidatedRow) error {
	var aug augmentedParentInviteRow
	if err := json.Unmarshal(row.RawData, &aug); err != nil {
		return &imports.ImportError{
			Type:    imports.ImportFailureSchemaValidation,
			Message: fmt.Sprintf("unmarshal augmented row: %v", err),
		}
	}

	if pi.stytch == nil {
		return &imports.ImportError{
			Type:    imports.ImportFailureStytchAPIError,
			Message: "authentication provider not configured",
		}
	}

	// Step 1: Create Stytch member and send invitation email.
	inviteCallbackURL := pi.backendURL + ParentInviteRedirectPath
	stytchMemberID, err := pi.stytch.InviteMemberByEmail(ctx, aug.StytchOrgID, aug.Email, aug.FullName, inviteCallbackURL)
	if err != nil {
		return &imports.ImportError{
			Type:    imports.ImportFailureStytchAPIError,
			Message: fmt.Sprintf("Could not send invitation email: %v", err),
		}
	}

	// Step 2: Insert invitation record in DB with role=PARENT.
	params := InsertInvitationParams{
		Email:          aug.Email,
		FullName:       aug.FullName,
		TenantID:       uuid.MustParse(aug.TenantID),
		SchoolID:       uuid.MustParse(aug.SchoolID),
		Role:           "PARENT",
		InvitedBy:      mustParseUUID(aug.InvitedBy),
		Status:         "pending",
		StytchMemberID: stytchMemberID,
		ExpiresAt:      time.Now().Add(parentInviteExpiryDays * 24 * time.Hour),
		ImportJobID:    mustParseUUID(aug.ImportJobID),
	}

	if err := pi.repo.InsertInvitation(ctx, tx, params); err != nil {
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

// compile-time interface check
var _ imports.Importer = (*ParentInviteImporter)(nil)
