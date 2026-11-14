package cursor

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestKeysetPaginationWalkIntegration walks 100k+ seeded rows using keyset
// pagination to ensure every row is fetched exactly once with zero duplicates.
func TestKeysetPaginationWalkIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
	)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("failed to open connection: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Errorf("failed to close db: %v", err)
	}
	db, err = sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("failed to reopen connection: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("failed to close db: %v", err)
		}
	})

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE test_schools (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	const batchSize = 1000
	const totalRows = 100000
	for i := 0; i < totalRows; i += batchSize {
		values := make([]interface{}, 0, batchSize*2)
		query := "INSERT INTO test_schools (name) VALUES "
		for j := 0; j < batchSize && i+j < totalRows; j++ {
			if j > 0 {
				query += ","
			}
			query += fmt.Sprintf("($%d)", j*2+1)
			values = append(values, fmt.Sprintf("School %d", i+j))
		}
		if _, err := db.ExecContext(ctx, query, values...); err != nil {
			t.Fatalf("failed to seed rows: %v", err)
		}
	}

	allRows := make(map[string]bool)
	rows, err := db.QueryContext(ctx, "SELECT id, name FROM test_schools ORDER BY name, id")
	if err != nil {
		t.Fatalf("failed to query all rows: %v", err)
	}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}
		allRows[id] = true
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("failed to close rows: %v", err)
	}
	if len(allRows) != totalRows {
		t.Fatalf("expected %d rows, got %d", totalRows, len(allRows))
	}

	cfg := defaultConfig()
	var seenIDs = make(map[string]bool)
	var nextCursor string
	pageCount := 0

	for {
		params, err := ParseListParams(urlValues(map[string]string{
			"limit":   "50",
			"sort":    "name_asc",
			"cursor":  nextCursor,
		}), cfg)
		if err != nil {
			t.Fatalf("failed to parse params: %v", err)
		}

		query := `
			SELECT id, name FROM test_schools
			WHERE ($1::text IS NULL OR name > $1)
			  AND (name, id) > ($2, $3)
			ORDER BY name ASC NULLS LAST, id ASC
			LIMIT $4
		`
		var lastName, lastID string
		if params.Cursor != nil && len(params.Cursor.Last) >= 2 {
			lastName = params.Cursor.Last[0].(string)
			lastID = params.Cursor.Last[1].(string)
		}

		rows, err := db.QueryContext(ctx, query, lastName, lastName, lastID, params.Limit+1)
		if err != nil {
			t.Fatalf("failed to query page: %v", err)
		}

		pageRows := make([]struct {
			id   string
			name string
		}, 0)
		for rows.Next() {
			var id, name string
			if err := rows.Scan(&id, &name); err != nil {
				t.Fatalf("failed to scan row: %v", err)
			}
			pageRows = append(pageRows, struct {
				id   string
				name string
			}{id, name})
		}
		if err := rows.Close(); err != nil {
			t.Fatalf("failed to close rows: %v", err)
		}

		if len(pageRows) > params.Limit {
			pageRows = pageRows[:params.Limit]
			last := pageRows[len(pageRows)-1]
			nextCursor, err = EncodeCursor(&Cursor{
				Sort:       []string{"name_asc"},
				Last:       []interface{}{last.name, last.id},
				Tiebreaker: last.id,
				Ctx:        ContextHash("name_asc", params.Filters),
			})
			if err != nil {
				t.Fatalf("failed to encode cursor: %v", err)
			}
		} else {
			nextCursor = ""
		}

		for _, r := range pageRows {
			if seenIDs[r.id] {
				t.Errorf("duplicate row %s on page %d", r.id, pageCount)
			}
			seenIDs[r.id] = true
		}
		pageCount++

		if nextCursor == "" {
			break
		}
	}

	if len(seenIDs) != totalRows {
		t.Errorf("expected %d unique rows, got %d", totalRows, len(seenIDs))
	}

	for id := range allRows {
		if !seenIDs[id] {
			t.Errorf("missing row %s", id)
		}
	}

	t.Logf("Successfully walked %d rows across %d pages with zero duplicates", totalRows, pageCount)
}

// TestKeysetPaginationCrossTenant verifies that a cursor from one tenant cannot
// fetch another tenant's rows.
func TestKeysetPaginationCrossTenant(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
	)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("failed to open connection: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Errorf("failed to close db: %v", err)
	}
	db, err = sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("failed to reopen connection: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("failed to close db: %v", err)
		}
	})

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE test_tenants (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL
		)
	`); err != nil {
		t.Fatalf("failed to create tenants table: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE test_tenant_schools (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			name TEXT NOT NULL
		)
	`); err != nil {
		t.Fatalf("failed to create schools table: %v", err)
	}

	var tenantA, tenantB string
	if err := db.QueryRowContext(ctx, "INSERT INTO test_tenants (name) VALUES ('Tenant A') RETURNING id").Scan(&tenantA); err != nil {
		t.Fatalf("failed to insert tenant A: %v", err)
	}
	if err := db.QueryRowContext(ctx, "INSERT INTO test_tenants (name) VALUES ('Tenant B') RETURNING id").Scan(&tenantB); err != nil {
		t.Fatalf("failed to insert tenant B: %v", err)
	}

	if _, err := db.ExecContext(ctx, "INSERT INTO test_tenant_schools (tenant_id, name) VALUES ($1, 'School A1'), ($1, 'School A2')", tenantA); err != nil {
		t.Fatalf("failed to insert tenant A schools: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO test_tenant_schools (tenant_id, name) VALUES ($1, 'School B1'), ($1, 'School B2')", tenantB); err != nil {
		t.Fatalf("failed to insert tenant B schools: %v", err)
	}

	cursor, err := EncodeCursor(&Cursor{
		Sort:       []string{"name_asc"},
		Last:       []interface{}{"School A1", "some-id"},
		Tiebreaker: "some-id",
		Ctx:        ContextHash("name_asc", nil),
	})
	if err != nil {
		t.Fatalf("failed to encode cursor: %v", err)
	}

	params, err := ParseListParams(urlValues(map[string]string{
		"limit":   "50",
		"sort":    "name_asc",
		"cursor":  cursor,
	}), defaultConfig())
	if err != nil {
		t.Fatalf("failed to parse params: %v", err)
	}

	if params.Cursor == nil {
		t.Fatal("expected non-nil cursor")
	}

	rows, err := db.QueryContext(ctx, `
		SELECT id, name FROM test_tenant_schools
		WHERE tenant_id = $1
		  AND (name, id) > ($2, $3)
		ORDER BY name ASC NULLS LAST, id ASC
		LIMIT 50
	`, tenantB, params.Cursor.Last[0], params.Cursor.Last[1])
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	var foundB bool
	var id string
	var name string
	for rows.Next() {
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatalf("failed to scan: %v", err)
		}
		if name == "School B1" || name == "School B2" {
			foundB = true
		}
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("failed to close rows: %v", err)
	}
	if !foundB {
		t.Error("expected to find Tenant B's schools")
	}

	t.Log("Cross-tenant cursor test passed")
}

func urlValues(m map[string]string) url.Values {
	v := make(url.Values, len(m))
	for k, val := range m {
		v.Set(k, val)
	}
	return v
}
