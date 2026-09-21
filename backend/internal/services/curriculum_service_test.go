package services

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"somotracker/backend/internal/database/sqlc"
	"somotracker/backend/internal/testdb"
)

func TestCurriculumService_ListSubjects_Integration(t *testing.T) {
	testdb.Setup(t)
	defer testdb.Teardown(t)
	ctx := context.Background()
	queries := sqlc.New(testdb.TestPool)
	_ = testdb.LoadSeeds(ctx, testdb.TestPool)

	svc := NewCurriculumService(queries, zap.NewNop())
	items, total, err := svc.ListSubjects(ctx, 1, 10, "", "")
	if err != nil {
		t.Fatalf("ListSubjects error: %v", err)
	}
	if items == nil {
		t.Fatalf("expected items not nil")
	}
	t.Logf("got %d subjects, total %d", len(items), total)
}
