//go:build integration

package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/testdb"
)

func TestStudentImportService_CreateBulkJobWithItems_Idempotency(t *testing.T) {
	ctx := context.Background()
	db := testdb.DB(t)
	svc := NewStudentImportService(db.Pool(), zap.NewNop())

	schoolID := uuid.New()
	tenantID := uuid.New()
	createdBy := uuid.New()
	key := "idem-key-" + uuid.NewString()

	items := []StudentImportItem{
		{AdmissionNumber: "A001", FullName: "Alice", DateOfBirth: "2006-01-01", Gender: "F"},
		{AdmissionNumber: "A002", FullName: "Bob", DateOfBirth: "2006-02-02", Gender: "M"},
	}

	jobID1, err := svc.CreateBulkJobWithItems(ctx, schoolID, tenantID, createdBy, key, items)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, jobID1)

	jobID2, err := svc.CreateBulkJobWithItems(ctx, schoolID, tenantID, createdBy, key, items)
	require.NoError(t, err)
	require.Equal(t, jobID1, jobID2, "idempotent call should return same job id")

	// Verify items count not duplicated
	jobItems, err := svc.GetItemsByJobID(ctx, jobID1)
	require.NoError(t, err)
	require.Len(t, jobItems, 2)
}

func TestStudentImportService_TryFinalizeJob_Atomic(t *testing.T) {
	ctx := context.Background()
	db := testdb.DB(t)
	svc := NewStudentImportService(db.Pool(), zap.NewNop())

	schoolID := uuid.New()
	tenantID := uuid.New()
	createdBy := uuid.New()
	key := "finalize-key-" + uuid.NewString()

	items := []StudentImportItem{
		{AdmissionNumber: "B001", FullName: "Carol", DateOfBirth: "2006-03-03", Gender: "F"},
	}
	jobID, err := svc.CreateBulkJobWithItems(ctx, schoolID, tenantID, createdBy, key, items)
	require.NoError(t, err)

	// Simulate counters not complete
	finalized, err := svc.TryFinalizeJob(ctx, jobID)
	require.NoError(t, err)
	require.False(t, finalized, "should not finalize while counters incomplete")

	// Now increment counters to match total
	err = svc.IncrementJobCounters(ctx, jobID, 1, 0, 0)
	require.NoError(t, err)

	finalized, err = svc.TryFinalizeJob(ctx, jobID)
	require.NoError(t, err)
	require.True(t, finalized, "should finalize when counters match total")
}
