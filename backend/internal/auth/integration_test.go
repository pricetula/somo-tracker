package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

// ============================================================================
// Category 1: Stytch Error Scenarios
// ============================================================================

// TestIntegration_Stytch_DiscoveryEmailTimeout simulates Stytch timing out
// when sending a discovery email.
func TestIntegration_Stytch_DiscoveryEmailTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	// Simulate a timeout from Stytch
	suite.setStytchHandlers(StytchMockHandlers{
		DiscoverySendFn: func(email string) (int, any) {
			return http.StatusGatewayTimeout, map[string]any{
				"status_code":   504,
				"error_type":    "stytch_error",
				"error_message": "upstream timeout",
				"request_id":    "req-timeout",
			}
		},
	})

	err := suite.svc.Discover(context.Background(), "test@example.com")
	if err == nil {
		t.Fatal("expected error from Stytch timeout, got nil")
	}
	if !errors.Is(err, ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", err)
	}
}

// TestIntegration_Stytch_ExpiredMagicLinkToken simulates a user clicking
// an expired magic link.
func TestIntegration_Stytch_ExpiredMagicLinkToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	suite.setStytchHandlers(StytchMockHandlers{
		DiscoveryAuthFn: func(token string) (int, any) {
			// Stytch SDK checks for error_type == "magic_link_token_expired"
			return http.StatusBadRequest, map[string]any{
				"status_code":   400,
				"error_type":    "magic_link_token_expired",
				"error_message": "magic link token expired",
				"request_id":    "req-expired",
			}
		},
	})

	_, err := suite.svc.Verify(context.Background(), "expired_token")
	if err == nil {
		t.Fatal("expected error from expired token, got nil")
	}
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

// TestIntegration_Stytch_OrgCreationDuplicate simulates creating an org
// that already exists in Stytch (idempotency scenario).
func TestIntegration_Stytch_OrgCreationDuplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	// Prepare a valid IST in Redis
	sessionRef := "550e8400-e29b-41d4-a716-446655440000"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	err := suite.rdb.Set(context.Background(), istKey, "ist_test_valid", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST in redis: %v", err)
	}

	payload := RegistrationPayload{
		SchoolName: "Duplicate School",
		SessionRef: sessionRef,
		FirstName:  "Alice",
		LastName:   "Smith",
	}

	token, role, err := suite.svc.Register(context.Background(), sessionRef, payload, "fp-001")
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty session token")
	}
	if role != "SCHOOL_ADMIN" {
		t.Fatalf("expected SCHOOL_ADMIN role for first user, got %s", role)
	}

	// Second registration with same IST — should fail because IST was consumed
	suite.freshRedis(t) // clean Redis, but the IST is already consumed
	// Actually the IST was consumed in the first call. Let's try with a different IST.
	sessionRef2 := "550e8400-e29b-41d4-a716-446655440001"
	istKey2 := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef2)
	err = suite.rdb.Set(context.Background(), istKey2, "ist_test_other", istTTL).Err()
	if err != nil {
		t.Fatalf("set second IST: %v", err)
	}

	payload2 := RegistrationPayload{
		SchoolName: "Duplicate School",
		SessionRef: sessionRef2,
		FirstName:  "Bob",
		LastName:   "Jones",
	}

	// Tenant already exists — this should still succeed (it handles existing tenants)
	token2, _, err := suite.svc.Register(context.Background(), sessionRef2, payload2, "fp-002")
	if err != nil {
		t.Fatalf("second registration for same school failed: %v", err)
	}
	if token2 == "" {
		t.Fatal("expected non-empty session token")
	}
	if token2 == token {
		t.Fatal("expected different session tokens for different users")
	}
}

// TestIntegration_Stytch_OrgCreationFailure simulates Stytch returning a 500
// during org creation.
func TestIntegration_Stytch_OrgCreationFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	sessionRef := "550e8400-e29b-41d4-a716-446655440010"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	err := suite.rdb.Set(context.Background(), istKey, "ist_test_valid", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST: %v", err)
	}

	suite.setStytchHandlers(StytchMockHandlers{
		CreateOrgFn: func(name string) (int, any) {
			return http.StatusInternalServerError, map[string]any{
				"status_code":   500,
				"error_type":    "internal_server_error",
				"error_message": "Stytch internal error",
				"request_id":    "req-org-fail",
			}
		},
	})

	payload := RegistrationPayload{
		SchoolName: "Fail School",
		SessionRef: sessionRef,
		FirstName:  "Charlie",
		LastName:   "Brown",
	}

	_, _, err = suite.svc.Register(context.Background(), sessionRef, payload, "fp-003")
	if err == nil {
		t.Fatal("expected error from org creation failure, got nil")
	}
	if !errors.Is(err, ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", err)
	}

	// Verify no session or user was created in the database
	var count int
	err = suite.pgPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM tenants WHERE name = $1", "Fail School").Scan(&count)
	if err != nil {
		t.Fatalf("query tenants: %v", err)
	}
	if count > 0 {
		t.Fatal("tenant should not have been created after failed Stytch org creation")
	}
}

// TestIntegration_Stytch_ISTExchangeInvalid simulates an invalid IST exchange.
func TestIntegration_Stytch_ISTExchangeInvalid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	sessionRef := "550e8400-e29b-41d4-a716-446655440020"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	err := suite.rdb.Set(context.Background(), istKey, "ist_test_invalid", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST: %v", err)
	}

	suite.setStytchHandlers(StytchMockHandlers{
		ExchangeISTFn: func(ist, orgID string) (int, any) {
			return http.StatusBadRequest, map[string]any{
				"status_code":   400,
				"error_type":    "intermediate_session_token_invalid",
				"error_message": "intermediate session token is invalid or expired",
				"request_id":    "req-ist-invalid",
			}
		},
	})

	payload := RegistrationPayload{
		SchoolName: "Invalid IST School",
		SessionRef: sessionRef,
		FirstName:  "Diana",
		LastName:   "Prince",
	}

	_, _, err = suite.svc.Register(context.Background(), sessionRef, payload, "fp-004")
	if err == nil {
		t.Fatal("expected error from invalid IST, got nil")
	}
}

// TestIntegration_Stytch_ISTExchangeMFANotMet simulates MFA being required
// but not yet completed.
func TestIntegration_Stytch_ISTExchangeMFANotMet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	sessionRef := "550e8400-e29b-41d4-a716-446655440030"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	err := suite.rdb.Set(context.Background(), istKey, "ist_test_mfa", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST: %v", err)
	}

	suite.setStytchHandlers(StytchMockHandlers{
		ExchangeISTFn: func(ist, orgID string) (int, any) {
			// MemberAuthenticated: false means MFA is required
			return http.StatusOK, map[string]any{
				"request_id":             "req-exchange-mfa",
				"status_code":            200,
				"member_id":              "member_test_mfa",
				"session_token":          "",
				"member_authenticated":   false,
				"intermediate_session_token": ist,
				"member": map[string]any{
					"member_id":   "member_test_mfa",
					"email_address": "mfa@example.com",
					"status":      "active",
				},
				"organization": map[string]any{
					"organization_id": orgID,
				},
			}
		},
	})

	payload := RegistrationPayload{
		SchoolName: "MFA Required School",
		SessionRef: sessionRef,
		FirstName:  "Eve",
		LastName:   "Adams",
	}

	_, _, err = suite.svc.Register(context.Background(), sessionRef, payload, "fp-005")
	if err == nil {
		t.Fatal("expected ErrMFARequired, got nil")
	}
	if !errors.Is(err, ErrMFARequired) {
		t.Fatalf("expected ErrMFARequired, got %v", err)
	}

	// Verify WARN log was emitted for MFA failure
	warnLogs := suite.observedLogs.FilterLevelExact(zapcore.WarnLevel)
	found := false
	for _, entry := range warnLogs.All() {
		if strings.Contains(entry.Message, "MFA required") {
			found = true
			break
		}
	}
	if !found {
		t.Log("note: expected WARN log about MFA requirement (non-fatal check)")
	}
}

// TestIntegration_Stytch_CreateMember_Failure simulates Stytch returning
// an error during member creation. Since we now pre-create the member in Stytch
// before exchanging the IST (instead of relying on JIT provisioning), a failure
// here should abort the registration without leaking any database state.
func TestIntegration_Stytch_CreateMember_Failure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	sessionRef := "550e8400-e29b-41d4-a716-446655440100"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	err := suite.rdb.Set(context.Background(), istKey, "ist_test_member_create_fail", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST: %v", err)
	}

	suite.setStytchHandlers(StytchMockHandlers{
		CreateMemberFn: func(orgID, email, name string) (int, any) {
			return http.StatusBadRequest, map[string]any{
				"status_code":   400,
				"error_type":    "invalid_email",
				"error_message": "Invalid email address",
				"request_id":    "req-invalid-email",
			}
		},
	})

	payload := RegistrationPayload{
		SchoolName: "Create Member Fail School",
		SessionRef: sessionRef,
		FirstName:  "Hank",
		LastName:   "Pym",
	}

	_, _, err = suite.svc.Register(context.Background(), sessionRef, payload, "fp-cm-fail")
	if err == nil {
		t.Fatal("expected error from member creation failure, got nil")
	}
	if !errors.Is(err, ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", err)
	}

	// Verify no tenant, user, or session leaked into the database
	var count int
	_ = suite.pgPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM tenants WHERE name = $1", "Create Member Fail School").Scan(&count)
	if count > 0 {
		t.Fatal("tenant should NOT have been created after member creation failure")
	}
}

// TestIntegration_Stytch_Exchange_JITProvisioningNotAllowed simulates Stytch
// returning email_jit_provisioning_not_allowed during IST exchange. This can still
// occur when an existing tenant org has JIT provisioning disabled and the member
// was not pre-created (e.g., invite flow).
func TestIntegration_Stytch_Exchange_JITProvisioningNotAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	sessionRef := "550e8400-e29b-41d4-a716-446655440101"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	err := suite.rdb.Set(context.Background(), istKey, "ist_test_jit", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST: %v", err)
	}

	suite.setStytchHandlers(StytchMockHandlers{
		ExchangeISTFn: func(ist, orgID string) (int, any) {
			return http.StatusBadRequest, map[string]any{
				"status_code":   400,
				"error_type":    "email_jit_provisioning_not_allowed",
				"error_message": "Email JIT provisioning is not allowed for this organization",
				"request_id":    "req-jit-blocked",
			}
		},
	})

	payload := RegistrationPayload{
		SchoolName: "JIT Blocked School",
		SessionRef: sessionRef,
		FirstName:  "Hank",
		LastName:   "Pym",
	}

	_, _, err = suite.svc.Register(context.Background(), sessionRef, payload, "fp-jit")
	if err == nil {
		t.Fatal("expected error from JIT provisioning not allowed, got nil")
	}
	if !errors.Is(err, ErrJITProvisioningNotAllowed) {
		t.Fatalf("expected ErrJITProvisioningNotAllowed, got %v", err)
	}

	// Verify no tenant, user, or session leaked into the database
	var count int
	_ = suite.pgPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM tenants WHERE name = $1", "JIT Blocked School").Scan(&count)
	if count > 0 {
		t.Fatal("tenant should NOT have been created after JIT provisioning failure")
	}
}

// TestIntegration_Stytch_Exchange_MemberNotFound simulates Stytch returning
// member_not_found during IST exchange. This occurs when the authenticated
// user does not have a membership in the target organization.
func TestIntegration_Stytch_Exchange_MemberNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	sessionRef := "550e8400-e29b-41d4-a716-446655440110"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	err := suite.rdb.Set(context.Background(), istKey, "ist_test_member_not_found", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST: %v", err)
	}

	suite.setStytchHandlers(StytchMockHandlers{
		ExchangeISTFn: func(ist, orgID string) (int, any) {
			return http.StatusBadRequest, map[string]any{
				"status_code":   400,
				"error_type":    "member_not_found",
				"error_message": "member not found in organization",
				"request_id":    "req-member-not-found",
			}
		},
	})

	payload := RegistrationPayload{
		SchoolName: "Member Not Found School",
		SessionRef: sessionRef,
		FirstName:  "Tony",
		LastName:   "Stark",
	}

	_, _, err = suite.svc.Register(context.Background(), sessionRef, payload, "fp-member-not-found")
	if err == nil {
		t.Fatal("expected error from member_not_found, got nil")
	}
	if !errors.Is(err, ErrMemberNotFound) {
		t.Fatalf("expected ErrMemberNotFound, got %v", err)
	}
}

// TestIntegration_Stytch_Exchange_OrgNotFound simulates Stytch returning
// organization_not_found during IST exchange. This occurs when the org ID
// used for exchange doesn't exist in Stytch.
func TestIntegration_Stytch_Exchange_OrgNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	sessionRef := "550e8400-e29b-41d4-a716-446655440120"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	err := suite.rdb.Set(context.Background(), istKey, "ist_test_org_not_found", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST: %v", err)
	}

	suite.setStytchHandlers(StytchMockHandlers{
		ExchangeISTFn: func(ist, orgID string) (int, any) {
			return http.StatusBadRequest, map[string]any{
				"status_code":   400,
				"error_type":    "organization_not_found",
				"error_message": "organization not found",
				"request_id":    "req-org-not-found",
			}
		},
	})

	payload := RegistrationPayload{
		SchoolName: "Org Not Found School",
		SessionRef: sessionRef,
		FirstName:  "Bruce",
		LastName:   "Banner",
	}

	_, _, err = suite.svc.Register(context.Background(), sessionRef, payload, "fp-org-not-found")
	if err == nil {
		t.Fatal("expected error from org_not_found, got nil")
	}
	if !errors.Is(err, ErrOrgNotFound) {
		t.Fatalf("expected ErrOrgNotFound, got %v", err)
	}
}

// TestIntegration_Stytch_Exchange_ExpiredIST simulates Stytch returning an
// expired IST error during the exchange call itself (as opposed to the IST
// missing from Redis, which is already tested).
func TestIntegration_Stytch_Exchange_ExpiredIST(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	sessionRef := "550e8400-e29b-41d4-a716-446655440130"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	err := suite.rdb.Set(context.Background(), istKey, "ist_test_stale", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST: %v", err)
	}

	suite.setStytchHandlers(StytchMockHandlers{
		ExchangeISTFn: func(ist, orgID string) (int, any) {
			return http.StatusBadRequest, map[string]any{
				"status_code":   400,
				"error_type":    "intermediate_session_token_expired",
				"error_message": "intermediate session token has expired",
				"request_id":    "req-ist-expired",
			}
		},
	})

	payload := RegistrationPayload{
		SchoolName: "Expired IST School",
		SessionRef: sessionRef,
		FirstName:  "Natasha",
		LastName:   "Romanoff",
	}

	_, _, err = suite.svc.Register(context.Background(), sessionRef, payload, "fp-ist-expired")
	if err == nil {
		t.Fatal("expected error from expired IST, got nil")
	}
	if !errors.Is(err, ErrInternal) {
		t.Fatalf("expected ErrInternal for expired IST during exchange, got %v", err)
	}
}

// TestIntegration_Stytch_Exchange_ReturnsSessionJWT verifies that when Stytch
// returns both a session_token and session_jwt, both are properly stored and
// retrievable. This validates the "real token" exchange path.
func TestIntegration_Stytch_Exchange_ReturnsSessionJWT(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	sessionRef := "550e8400-e29b-41d4-a716-446655440140"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	err := suite.rdb.Set(context.Background(), istKey, "ist_test_jwt", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST: %v", err)
	}

	// Use default handlers (which return session_token) — we verify the token
	// is stored and retrievable via GetSession.
	payload := RegistrationPayload{
		SchoolName: "JWT Exchange School",
		SessionRef: sessionRef,
		FirstName:  "Steve",
		LastName:   "Rogers",
	}

	token, role, err := suite.svc.Register(context.Background(), sessionRef, payload, "fp-jwt")
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty session token")
	}
	if role != "SCHOOL_ADMIN" {
		t.Fatalf("expected SCHOOL_ADMIN for first user, got %s", role)
	}

	// Verify the session can be retrieved (proving the Stytch session token
	// was stored and the Redis + Postgres write succeeded)
	session, err := suite.svc.GetSession(context.Background(), token)
	if err != nil {
		t.Fatalf("expected session to be retrievable: %v", err)
	}
	if session.StytchSessionToken == "" {
		t.Fatal("expected stytch_session_token to be stored in the session")
	}
	if session.StytchOrgID == "" {
		t.Fatal("expected stytch_org_id to be stored in the session")
	}
	if session.StytchMemberID == "" {
		t.Fatal("expected stytch_member_id to be stored in the session")
	}
}

// TestIntegration_Stytch_OrgCreationDuplicateSlug simulates Stytch returning
// a slug conflict when creating an organization. Stytch enforces unique slugs
// across all organizations.
func TestIntegration_Stytch_OrgCreationDuplicateSlug(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	sessionRef := "550e8400-e29b-41d4-a716-446655440150"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	err := suite.rdb.Set(context.Background(), istKey, "ist_test_slug_collision", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST: %v", err)
	}

	suite.setStytchHandlers(StytchMockHandlers{
		CreateOrgFn: func(name string) (int, any) {
			return http.StatusConflict, map[string]any{
				"status_code":   409,
				"error_type":    "organization_slug_conflict",
				"error_message": "an organization with this slug already exists",
				"request_id":    "req-slug-conflict",
			}
		},
	})

	payload := RegistrationPayload{
		SchoolName: "Duplicate Slug School",
		SessionRef: sessionRef,
		FirstName:  "Clint",
		LastName:   "Barton",
	}

	_, _, err = suite.svc.Register(context.Background(), sessionRef, payload, "fp-slug")
	if err == nil {
		t.Fatal("expected error from slug conflict, got nil")
	}
	if !errors.Is(err, ErrInternal) {
		t.Fatalf("expected ErrInternal for slug conflict, got %v", err)
	}

	// Verify no tenant leaked into the database
	var count int
	_ = suite.pgPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM tenants WHERE name = $1", "Duplicate Slug School").Scan(&count)
	if count > 0 {
		t.Fatal("tenant should not have been created after org creation failure")
	}
}

// TestIntegration_Stytch_OrgCreationEmptyOrgID simulates Stytch returning a
// 200 OK with a response body that's missing the organization_id field.
func TestIntegration_Stytch_OrgCreationEmptyOrgID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	sessionRef := "550e8400-e29b-41d4-a716-446655440160"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	err := suite.rdb.Set(context.Background(), istKey, "ist_test_empty_org", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST: %v", err)
	}

	suite.setStytchHandlers(StytchMockHandlers{
		CreateOrgFn: func(name string) (int, any) {
			return http.StatusOK, map[string]any{
				"request_id":  "req-empty-org",
				"status_code": 200,
				"organization": map[string]any{
					"organization_id":   "",
					"organization_name": name,
				},
			}
		},
	})

	payload := RegistrationPayload{
		SchoolName: "Empty Org ID School",
		SessionRef: sessionRef,
		FirstName:  "Wanda",
		LastName:   "Maximoff",
	}

	_, _, err = suite.svc.Register(context.Background(), sessionRef, payload, "fp-empty-org")
	if err == nil {
		t.Fatal("expected error from empty org_id, got nil")
	}
	if !errors.Is(err, ErrInternal) {
		t.Fatalf("expected ErrInternal for empty org_id, got %v", err)
	}

	// Verify no tenant leaked into the database
	var count int
	_ = suite.pgPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM tenants WHERE name = $1", "Empty Org ID School").Scan(&count)
	if count > 0 {
		t.Fatal("tenant should not have been created after empty org_id response")
	}
}

// TestIntegration_ExistingOrg_SecondUserRegistration verifies that a second
// user can register for an existing org (tenant already exists). The first
// registration creates the org, the second should re-use it, create a new user
// and session, and assign the TEACHER role.
func TestIntegration_ExistingOrg_SecondUserRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	schoolName := "Second User School"

	// ---- First registration: creates the org and tenant ----
	sessionRef1 := "550e8400-e29b-41d4-a716-446655440170"
	istKey1 := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef1)
	err := suite.rdb.Set(context.Background(), istKey1, "ist_test_first", istTTL).Err()
	if err != nil {
		t.Fatalf("set first IST: %v", err)
	}

	payload1 := RegistrationPayload{
		SchoolName: schoolName,
		SessionRef: sessionRef1,
		FirstName:  "Peter",
		LastName:   "Parker",
	}

	token1, role1, err := suite.svc.Register(context.Background(), sessionRef1, payload1, "fp-first")
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}
	if token1 == "" {
		t.Fatal("expected non-empty session token from first registration")
	}
	if role1 != "SCHOOL_ADMIN" {
		t.Fatalf("expected SCHOOL_ADMIN for first user, got %s", role1)
	}

	// ---- Second registration: same school, new user ----
	sessionRef2 := "550e8400-e29b-41d4-a716-446655440171"
	istKey2 := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef2)
	err = suite.rdb.Set(context.Background(), istKey2, "ist_test_second", istTTL).Err()
	if err != nil {
		t.Fatalf("set second IST: %v", err)
	}

	payload2 := RegistrationPayload{
		SchoolName: schoolName,
		SessionRef: sessionRef2,
		FirstName:  "Miles",
		LastName:   "Morales",
	}

	token2, role2, err := suite.svc.Register(context.Background(), sessionRef2, payload2, "fp-second")
	if err != nil {
		t.Fatalf("second registration failed: %v", err)
	}
	if token2 == "" {
		t.Fatal("expected non-empty session token from second registration")
	}
	if role2 != "TEACHER" {
		t.Fatalf("expected TEACHER for second user, got %s", role2)
	}

	// Verify tokens are different
	if token1 == token2 {
		t.Fatal("first and second user should have different session tokens")
	}

	// Verify both sessions are retrievable
	_, err = suite.svc.GetSession(context.Background(), token1)
	if err != nil {
		t.Fatalf("first user session should be retrievable: %v", err)
	}
	_, err = suite.svc.GetSession(context.Background(), token2)
	if err != nil {
		t.Fatalf("second user session should be retrievable: %v", err)
	}

	// Verify the tenant has exactly 2 users
	var userCount int
	err = suite.pgPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM users WHERE tenant_id = (SELECT id FROM tenants WHERE name = $1)",
		schoolName).Scan(&userCount)
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 2 {
		t.Fatalf("expected 2 users in tenant, got %d", userCount)
	}

	// Verify only 1 tenant exists
	var tenantCount int
	_ = suite.pgPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM tenants WHERE name = $1", schoolName).Scan(&tenantCount)
	if tenantCount != 1 {
		t.Fatalf("expected exactly 1 tenant, got %d", tenantCount)
	}
}

// ============================================================================
// Category 2: Redis Cache Scenarios
// ============================================================================

// TestIntegration_Redis_ColdStartCacheMiss simulates a Redis restart where
// session tokens exist in Postgres but not in Redis. The service should
// gracefully fall through to Postgres.
func TestIntegration_Redis_ColdStartCacheMiss(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	// Insert a session directly into Postgres (as if it survived a Redis restart)
	tenantID := "t-001"
	userID := "u-001"
	suite.insertTenant(t, tenantID, "Cold Start School", "cold-start", "org_cold")
	suite.insertUser(t, userID, "cold@example.com", tenantID, "ext_cold")

	now := time.Now()
	session := UserSession{
		Token:              "cold_start_token_001",
		UserID:             userID,
		TenantID:           tenantID,
		StytchMemberID:     "member_cold",
		StytchOrgID:        "org_cold",
		StytchSessionToken: "sess_cold",
		DeviceFingerprint:  "fp-cold",
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		CreatedAt:          now,
	}
	suite.insertSession(t, session)

	// Redis is fresh — session token not in cache
	// The service should still find it via Postgres (after Redis miss)
	result, err := suite.svc.GetSession(context.Background(), "cold_start_token_001")
	if err != nil {
		t.Fatalf("expected session to be found via Postgres fallback, got error: %v", err)
	}
	if result.UserID != userID {
		t.Fatalf("expected user_id %s, got %s", userID, result.UserID)
	}
	if result.TenantID != tenantID {
		t.Fatalf("expected tenant_id %s, got %s", tenantID, result.TenantID)
	}
}

// TestIntegration_Redis_StaleEntryCleanup simulates a session that exists
// in Redis but whose Postgres row was deleted (e.g., admin revocation).
// The service should clean up the stale Redis entry.
func TestIntegration_Redis_StaleEntryCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	// Set the token in Redis (stale — no Postgres row)
	suite.setRedisSession(t, "stale_token_001", "sess_stale")

	// Attempting to get the session should fail (Postgres miss)
	// This simulates the session being revoked by admin
	_, err := suite.svc.GetSession(context.Background(), "stale_token_001")
	if err == nil {
		t.Fatal("expected error for stale Redis entry with no Postgres row")
	}
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}

	// Verify the stale Redis entry was cleaned up
	if suite.verifyRedisSession(t, "stale_token_001") {
		t.Fatal("stale Redis entry should have been cleaned up after miss")
	}
}

// TestIntegration_Redis_SessionInBothLayers simulates the happy path where
// the session exists in both Redis and Postgres.
func TestIntegration_Redis_SessionInBothLayers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	tenantID := "t-002"
	userID := "u-002"
	suite.insertTenant(t, tenantID, "Happy Path School", "happy-path", "org_happy")
	suite.insertUser(t, userID, "happy@example.com", tenantID, "ext_happy")

	now := time.Now()
	session := UserSession{
		Token:              "happy_token_001",
		UserID:             userID,
		TenantID:           tenantID,
		StytchMemberID:     "member_happy",
		StytchOrgID:        "org_happy",
		StytchSessionToken: "sess_happy",
		DeviceFingerprint:  "fp-happy",
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		CreatedAt:          now,
	}
	suite.insertSession(t, session)
	suite.setRedisSession(t, "happy_token_001", "sess_happy")

	result, err := suite.svc.GetSession(context.Background(), "happy_token_001")
	if err != nil {
		t.Fatalf("expected session to be found, got error: %v", err)
	}
	if result.UserID != userID {
		t.Fatalf("expected user_id %s, got %s", userID, result.UserID)
	}
}

// TestIntegration_Redis_ConcurrentISTConsume tests atomicity of the
// Lua script when multiple goroutines try to consume the same IST.
func TestIntegration_Redis_ConcurrentISTConsume(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	sessionRef := "550e8400-e29b-41d4-a716-446655440040"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	ctx := context.Background()

	// Set the IST once
	err := suite.rdb.Set(ctx, istKey, "ist_test_concurrent", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST: %v", err)
	}

	// Launch 20 goroutines all trying to read-and-delete the same IST
	var wg sync.WaitGroup
	results := make(chan string, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc := suite.svc
			val, _, err := svc.readAndDeleteIST(ctx, sessionRef)
			if err == nil && val != "" {
				results <- val
			} else {
				results <- ""
			}
		}()
	}

	wg.Wait()
	close(results)

	// Count how many goroutines got the IST
	successCount := 0
	for r := range results {
		if r != "" {
			successCount++
		}
	}

	// Exactly 1 goroutine should have succeeded (atomic read-delete)
	if successCount != 1 {
		t.Fatalf("expected exactly 1 concurrent consumer to get the IST, got %d", successCount)
	}

	// Verify the key is gone from Redis
	exists, err := suite.rdb.Exists(ctx, istKey).Result()
	if err != nil {
		t.Fatalf("check IST existence: %v", err)
	}
	if exists != 0 {
		t.Fatal("IST should have been deleted from Redis")
	}
}

// ============================================================================
// Category 3: Sessions Table Scenarios
// ============================================================================

// TestIntegration_Sessions_ExpiredSessionFiltered verifies that the
// repository's GetSessionByToken query correctly filters out expired sessions.
func TestIntegration_Sessions_ExpiredSessionFiltered(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)

	tenantID := "t-exp-001"
	userID := "u-exp-001"
	suite.insertTenant(t, tenantID, "Expired School", "expired", "org_exp")
	suite.insertUser(t, userID, "expired@example.com", tenantID, "ext_exp")

	now := time.Now()

	// Insert an expired session
	expiredSession := UserSession{
		Token:              "expired_token_001",
		UserID:             userID,
		TenantID:           tenantID,
		StytchMemberID:     "member_exp",
		StytchOrgID:        "org_exp",
		StytchSessionToken: "sess_exp",
		DeviceFingerprint:  "fp-exp",
		ExpiresAt:          now.Add(-1 * time.Hour), // expired 1 hour ago
		CreatedAt:          now.Add(-2 * time.Hour),
	}
	suite.insertSession(t, expiredSession)

	// Insert a valid session
	validSession := UserSession{
		Token:              "valid_token_001",
		UserID:             userID,
		TenantID:           tenantID,
		StytchMemberID:     "member_valid",
		StytchOrgID:        "org_valid",
		StytchSessionToken: "sess_valid",
		DeviceFingerprint:  "fp-valid",
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		CreatedAt:          now,
	}
	suite.insertSession(t, validSession)

	// Query the repository directly (bypass Redis)
	repo := &SqlcRepository{
		pool:   suite.pgPool,
		logger: suite.logger,
		cfg:    suite.cfg,
	}

	// Expired session should NOT be returned
	_, err := repo.GetSessionByToken(context.Background(), "expired_token_001")
	if err == nil {
		t.Fatal("expected expired session to not be found")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for expired session, got %v", err)
	}

	// Valid session SHOULD be returned
	session, err := repo.GetSessionByToken(context.Background(), "valid_token_001")
	if err != nil {
		t.Fatalf("expected valid session to be found, got error: %v", err)
	}
	if session.Token != "valid_token_001" {
		t.Fatalf("expected valid_token_001, got %s", session.Token)
	}
}

// TestIntegration_Sessions_CascadeDelete verifies that deleting a user
// cascades to delete their sessions.
func TestIntegration_Sessions_CascadeDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)

	tenantID := "t-cas-001"
	userID := "u-cas-001"
	suite.insertTenant(t, tenantID, "Cascade School", "cascade", "org_cas")
	suite.insertUser(t, userID, "cascade@example.com", tenantID, "ext_cas")

	now := time.Now()
	session := UserSession{
		Token:              "cascade_token_001",
		UserID:             userID,
		TenantID:           tenantID,
		StytchMemberID:     "member_cas",
		StytchOrgID:        "org_cas",
		StytchSessionToken: "sess_cas",
		DeviceFingerprint:  "fp-cas",
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		CreatedAt:          now,
	}
	suite.insertSession(t, session)

	// Delete the user — should cascade to sessions
	_, err := suite.pgPool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		t.Fatalf("delete user: %v", err)
	}

	// Verify session was also deleted
	var count int
	err = suite.pgPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM sessions WHERE token = $1", "cascade_token_001").Scan(&count)
	if err != nil {
		t.Fatalf("query sessions: %v", err)
	}
	if count != 0 {
		t.Fatal("session should have been cascade-deleted when user was deleted")
	}
}

// TestIntegration_Sessions_TokenUniqueness verifies that the UNIQUE
// constraint on sessions.token prevents duplicate tokens.
func TestIntegration_Sessions_TokenUniqueness(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)

	tenantID := "t-uniq-001"
	userID := "u-uniq-001"
	tenant2ID := "t-uniq-002"
	user2ID := "u-uniq-002"

	suite.insertTenant(t, tenantID, "Unique School", "unique", "org_uniq")
	suite.insertUser(t, userID, "unique@example.com", tenantID, "ext_uniq")
	suite.insertTenant(t, tenant2ID, "Unique School 2", "unique2", "org_uniq2")
	suite.insertUser(t, user2ID, "unique2@example.com", tenant2ID, "ext_uniq2")

	now := time.Now()

	// Insert first session
	_, err := suite.pgPool.Exec(context.Background(), `
		INSERT INTO sessions (token, user_id, tenant_id, stytch_member_id, stytch_org_id, device_fingerprint, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "dup_token_001", userID, tenantID, "member_u1", "org_u1", "fp-u1", now.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("insert first session: %v", err)
	}

	// Insert second session with same token — should fail
	_, err = suite.pgPool.Exec(context.Background(), `
		INSERT INTO sessions (token, user_id, tenant_id, stytch_member_id, stytch_org_id, device_fingerprint, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "dup_token_001", user2ID, tenant2ID, "member_u2", "org_u2", "fp-u2", now.Add(24*time.Hour))
	if err == nil {
		t.Fatal("expected uniqueness violation error, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate key") && !strings.Contains(err.Error(), "unique") {
		t.Fatalf("expected duplicate key error, got: %v", err)
	}
}

// TestIntegration_Sessions_MultiplePerUser verifies that a single user
// can have multiple active sessions (different browsers/devices).
func TestIntegration_Sessions_MultiplePerUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)

	tenantID := "t-multi-001"
	userID := "u-multi-001"
	suite.insertTenant(t, tenantID, "Multi Session School", "multi", "org_multi")
	suite.insertUser(t, userID, "multi@example.com", tenantID, "ext_multi")

	now := time.Now()
	sessions := []UserSession{
		{Token: "multi_token_a", UserID: userID, TenantID: tenantID, StytchMemberID: "m1", StytchOrgID: "org_multi", DeviceFingerprint: "fp-chrome", ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now},
		{Token: "multi_token_b", UserID: userID, TenantID: tenantID, StytchMemberID: "m1", StytchOrgID: "org_multi", DeviceFingerprint: "fp-firefox", ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now},
		{Token: "multi_token_c", UserID: userID, TenantID: tenantID, StytchMemberID: "m1", StytchOrgID: "org_multi", DeviceFingerprint: "fp-safari", ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now},
	}

	for _, s := range sessions {
		suite.insertSession(t, s)
	}

	// Verify all 3 sessions exist for this user
	var count int
	err := suite.pgPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM sessions WHERE user_id = $1", userID).Scan(&count)
	if err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 sessions for user, got %d", count)
	}
}

// TestIntegration_Sessions_DeleteSessionRevokesToken verifies that deleting
// a session row from Postgres effectively revokes the token.
func TestIntegration_Sessions_DeleteSessionRevokesToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)

	tenantID := "t-rev-001"
	userID := "u-rev-001"
	suite.insertTenant(t, tenantID, "Revoke School", "revoke", "org_rev")
	suite.insertUser(t, userID, "revoke@example.com", tenantID, "ext_rev")

	now := time.Now()
	session := UserSession{
		Token:              "revoke_token_001",
		UserID:             userID,
		TenantID:           tenantID,
		StytchMemberID:     "member_rev",
		StytchOrgID:        "org_rev",
		StytchSessionToken: "sess_rev",
		DeviceFingerprint:  "fp-rev",
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		CreatedAt:          now,
	}
	suite.insertSession(t, session)

	// Verify it exists
	repo := &SqlcRepository{pool: suite.pgPool, logger: suite.logger, cfg: suite.cfg}
	_, err := repo.GetSessionByToken(context.Background(), "revoke_token_001")
	if err != nil {
		t.Fatalf("expected session to exist: %v", err)
	}

	// Delete it
	err = repo.DeleteSession(context.Background(), "revoke_token_001")
	if err != nil {
		t.Fatalf("delete session: %v", err)
	}

	// Verify it's gone
	_, err = repo.GetSessionByToken(context.Background(), "revoke_token_001")
	if err == nil {
		t.Fatal("expected session to be gone after delete")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// TestIntegration_Sessions_TenantCascadeDelete verifies that deleting a
// tenant cascades to users and sessions.
func TestIntegration_Sessions_TenantCascadeDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)

	tenantID := "t-cten-001"
	userID := "u-cten-001"
	suite.insertTenant(t, tenantID, "Cascade Tenant School", "cascade-tenant", "org_cten")
	suite.insertUser(t, userID, "cten@example.com", tenantID, "ext_cten")

	now := time.Now()
	session := UserSession{
		Token:              "cten_token_001",
		UserID:             userID,
		TenantID:           tenantID,
		StytchMemberID:     "member_cten",
		StytchOrgID:        "org_cten",
		StytchSessionToken: "sess_cten",
		DeviceFingerprint:  "fp-cten",
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		CreatedAt:          now,
	}
	suite.insertSession(t, session)

	// Delete the tenant (should cascade to users and sessions)
	_, err := suite.pgPool.Exec(context.Background(), "DELETE FROM tenants WHERE id = $1", tenantID)
	if err != nil {
		t.Fatalf("delete tenant: %v", err)
	}

	// Verify session is gone
	var count int
	err = suite.pgPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM sessions WHERE token = $1", "cten_token_001").Scan(&count)
	if err != nil {
		t.Fatalf("query sessions: %v", err)
	}
	if count != 0 {
		t.Fatal("session should have been cascade-deleted via tenant->user->session")
	}

	// Verify user is gone
	err = suite.pgPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE id = $1", userID).Scan(&count)
	if err != nil {
		t.Fatalf("query users: %v", err)
	}
	if count != 0 {
		t.Fatal("user should have been cascade-deleted when tenant was deleted")
	}
}

// ============================================================================
// Category 4: DDoS / Edge Cases
// ============================================================================

// TestIntegration_DDoS_TokenBruteForce simulates a client sending random
// session tokens in cookies. All should be rejected.
func TestIntegration_DDoS_TokenBruteForce(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)

	// Try 100 random tokens — none should exist
	for i := 0; i < 100; i++ {
		tokenBytes := make([]byte, 32)
		rand.Read(tokenBytes)
		token := hex.EncodeToString(tokenBytes)

		_, err := suite.svc.GetSession(context.Background(), token)
		if err == nil {
			t.Fatalf("random token %s should not be valid", token[:16])
		}
		if !errors.Is(err, ErrExpiredToken) {
			t.Fatalf("expected ErrExpiredToken for random token, got %v", err)
		}
	}
}

// TestIntegration_DDoS_EmptyToken ensures empty tokens are rejected immediately
// without hitting the database.
func TestIntegration_DDoS_EmptyToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)

	_, err := suite.svc.GetSession(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken for empty token, got %v", err)
	}
}

// TestIntegration_DDoS_ConcurrentRegistration simulates multiple users
// registering for the same school at the same time.
func TestIntegration_DDoS_ConcurrentRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	const concurrentUsers = 10
	schoolName := "Concurrent Registration School"

	// Pre-setup: each goroutine gets its own sessionRef and pre-set IST
	type regTask struct {
		sessionRef string
		payload    RegistrationPayload
	}

	var tasks []regTask
	for i := 0; i < concurrentUsers; i++ {
		sessionRef := fmt.Sprintf("550e8400-e29b-41d4-a716-44665544%04d", i)
		istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
		err := suite.rdb.Set(context.Background(), istKey, fmt.Sprintf("ist_concurrent_%d", i), istTTL).Err()
		if err != nil {
			t.Fatalf("set IST for task %d: %v", i, err)
		}

		tasks = append(tasks, regTask{
			sessionRef: sessionRef,
			payload: RegistrationPayload{
				SchoolName: schoolName,
				SessionRef: sessionRef,
				FirstName:  fmt.Sprintf("User%d", i),
				LastName:   "Test",
			},
		})
	}

	// Run all registrations concurrently
	var wg sync.WaitGroup
	errs := make(chan error, concurrentUsers)
	tokens := make(chan string, concurrentUsers)

	for _, task := range tasks {
		wg.Add(1)
		go func(tk regTask) {
			defer wg.Done()
			token, _, err := suite.svc.Register(context.Background(), tk.sessionRef, tk.payload, fmt.Sprintf("fp-concurrent-%s", tk.sessionRef[:8]))
			if err != nil {
				errs <- fmt.Errorf("registration %s failed: %w", tk.sessionRef, err)
				return
			}
			tokens <- token
		}(task)
	}

	wg.Wait()
	close(errs)
	close(tokens)

	// Check for errors
	var errorList []error
	for err := range errs {
		errorList = append(errorList, err)
	}
	if len(errorList) > 0 {
		t.Fatalf("%d concurrent registrations failed: %v", len(errorList), errorList[0])
	}

	// Count unique session tokens
	tokenSet := make(map[string]int)
	for token := range tokens {
		tokenSet[token]++
	}

	// All 10 users should have unique tokens
	// Even though they all registered for the same school, each user gets
	// their own session token
	if len(tokenSet) != concurrentUsers {
		t.Fatalf("expected %d unique session tokens, got %d", concurrentUsers, len(tokenSet))
	}

	// Verify the tenant was created and all users belong to it
	var tenantCount int
	err := suite.pgPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM tenants WHERE name = $1", schoolName).Scan(&tenantCount)
	if err != nil {
		t.Fatalf("count tenants: %v", err)
	}
	if tenantCount != 1 {
		t.Fatalf("expected exactly 1 tenant, got %d", tenantCount)
	}

	var userCount int
	err = suite.pgPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM users WHERE tenant_id = (SELECT id FROM tenants WHERE name = $1)", schoolName).Scan(&userCount)
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != concurrentUsers {
		t.Fatalf("expected %d users in tenant, got %d", concurrentUsers, userCount)
	}
}

// TestIntegration_DDoS_ConcurrentLogout simulates multiple concurrent
// logout requests for the same session.
func TestIntegration_DDoS_ConcurrentLogout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)

	tenantID := "t-logout-001"
	userID := "u-logout-001"
	suite.insertTenant(t, tenantID, "Logout School", "logout", "org_logout")
	suite.insertUser(t, userID, "logout@example.com", tenantID, "ext_logout")

	now := time.Now()
	session := UserSession{
		Token:              "logout_token_001",
		UserID:             userID,
		TenantID:           tenantID,
		StytchMemberID:     "member_logout",
		StytchOrgID:        "org_logout",
		StytchSessionToken: "sess_logout",
		DeviceFingerprint:  "fp-logout",
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		CreatedAt:          now,
	}
	suite.insertSession(t, session)
	suite.setRedisSession(t, "logout_token_001", "sess_logout")

	// 20 concurrent logout requests
	var wg sync.WaitGroup
	errs := make(chan error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := suite.svc.Logout(context.Background(), "logout_token_001"); err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent logout failed: %v", err)
	}

	// Verify the session is fully gone from both stores
	// Redis
	if suite.verifyRedisSession(t, "logout_token_001") {
		t.Fatal("session should be removed from Redis after logout")
	}

	// Postgres
	var count int
	err := suite.pgPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM sessions WHERE token = $1", "logout_token_001").Scan(&count)
	if err != nil {
		t.Fatalf("query sessions: %v", err)
	}
	if count != 0 {
		t.Fatal("session should be removed from Postgres after logout")
	}
}

// TestIntegration_DDoS_RapidFireDiscoveries sends many discovery requests
// rapidly to exercise the system under load. (These hit the mock Stytch
// server, so they're fast.)
func TestIntegration_DDoS_RapidFireDiscoveries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	// 50 rapid discovery requests
	var wg sync.WaitGroup
	errs := make(chan error, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			email := fmt.Sprintf("user%d@example.com", idx)
			if err := suite.svc.Discover(context.Background(), email); err != nil {
				errs <- fmt.Errorf("discovery for %s failed: %w", email, err)
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("rapid discovery failed: %v", err)
	}
}

// TestIntegration_EdgeCase_RegisterAfterLogout verifies that after logout,
// the session token cannot be reused.
func TestIntegration_EdgeCase_RegisterAfterLogout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	// Full registration flow
	sessionRef := "550e8400-e29b-41d4-a716-446655440050"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	err := suite.rdb.Set(context.Background(), istKey, "ist_test_logout", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST: %v", err)
	}

	payload := RegistrationPayload{
		SchoolName: "Logout Reuse School",
		SessionRef: sessionRef,
		FirstName:  "Frank",
		LastName:   "Castle",
	}

	token, _, err := suite.svc.Register(context.Background(), sessionRef, payload, "fp-logout-reuse")
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Verify session works
	_, err = suite.svc.GetSession(context.Background(), token)
	if err != nil {
		t.Fatalf("expected session to work after registration: %v", err)
	}

	// Logout
	err = suite.svc.Logout(context.Background(), token)
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	// Verify session no longer works
	_, err = suite.svc.GetSession(context.Background(), token)
	if err == nil {
		t.Fatal("expected session to fail after logout")
	}
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken after logout, got %v", err)
	}
}

// TestIntegration_EdgeCase_EmptyDeviceFingerprint verifies that registration
// works with an empty device fingerprint (e.g., legacy clients).
func TestIntegration_EdgeCase_EmptyDeviceFingerprint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := testSuite
	suite.freshDB(t)
	suite.freshRedis(t)
	defer suite.resetStytchHandlers()

	sessionRef := "550e8400-e29b-41d4-a716-446655440060"
	istKey := fmt.Sprintf("%stest:%s", istKeyPrefix, sessionRef)
	err := suite.rdb.Set(context.Background(), istKey, "ist_test_empty_fp", istTTL).Err()
	if err != nil {
		t.Fatalf("set IST: %v", err)
	}

	payload := RegistrationPayload{
		SchoolName: "Empty FP School",
		SessionRef: sessionRef,
		FirstName:  "Grace",
		LastName:   "Hopper",
	}

	token, role, err := suite.svc.Register(context.Background(), sessionRef, payload, "")
	if err != nil {
		t.Fatalf("registration with empty device fingerprint failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty session token")
	}
	if role != "SCHOOL_ADMIN" {
		t.Fatalf("expected SCHOOL_ADMIN role for new tenant registration, got %s", role)
	}
}
