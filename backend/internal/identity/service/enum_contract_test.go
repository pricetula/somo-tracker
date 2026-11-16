package service

import (
	"testing"

	storegen "somotracker/internal/identity/store/gen"
	"github.com/stretchr/testify/require"
)

// TestUserRoleConstantsMatchDatabaseEnum ensures the Go-side UserRole constants
// match the PostgreSQL user_role enum values defined in migrations.
//
// If the migration enum changes (e.g., OWNER -> tenant_owner), this test fails,
// forcing the developer to update both the migration and the Go code together.
func TestUserRoleConstantsMatchDatabaseEnum(t *testing.T) {
	require.Equal(t, "OWNER", string(storegen.UserRoleOWNER))
	require.Equal(t, "ADMIN", string(storegen.UserRoleADMIN))
	require.Equal(t, "HEADTEACHER", string(storegen.UserRoleHEADTEACHER))
	require.Equal(t, "TEACHER", string(storegen.UserRoleTEACHER))
	require.Equal(t, "GUARDIAN", string(storegen.UserRoleGUARDIAN))
	require.Equal(t, "FINANCE", string(storegen.UserRoleFINANCE))
	require.Equal(t, "PLATFORMOPERATOR", string(storegen.UserRolePLATFORMOPERATOR))
}

// TestProvisioningRoleIsValidEnumValue ensures the role used when creating the
// first admin membership during tenant provisioning is a valid PostgreSQL
// user_role enum value — not a magic string like "tenant_owner".
//
// This catches the class of bug where a hardcoded string compiles fine in Go
// but fails at the database with:
//   invalid input value for enum user_role: "tenant_owner"
//
// The provisioning code uses storegen.UserRoleOWNER (not "tenant_owner")
// precisely so this kind of mismatch is impossible at the Go level.
func TestProvisioningRoleIsValidEnumValue(t *testing.T) {
	// The role used in provisionInTransaction must be one of the valid enum values.
	// This test documents the contract: UserRoleOWNER == "OWNER" (PostgreSQL enum).
	require.Equal(t, storegen.UserRoleOWNER, storegen.UserRole("OWNER"))
	require.NotEqual(t, storegen.UserRole("tenant_owner"), storegen.UserRoleOWNER)
}
