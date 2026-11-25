package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	storegen "somotracker/internal/auth/store/gen"
	"somotracker/internal/education/store"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/logger"
)

// mockStore implements store.StoreInterface using testify mock.
type mockEduStore struct {
	mock.Mock
}

func (m *mockEduStore) ListCountries(ctx context.Context) ([]storegen.Country, error) {
	args := m.Called(ctx)
	return args.Get(0).([]storegen.Country), args.Error(1)
}

func (m *mockEduStore) GetCountry(ctx context.Context, id uuid.UUID) (storegen.Country, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(storegen.Country), args.Error(1)
}

func (m *mockEduStore) ListEducationSystems(ctx context.Context) ([]storegen.ListEducationSystemsRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]storegen.ListEducationSystemsRow), args.Error(1)
}

func (m *mockEduStore) GetEducationSystem(ctx context.Context, id uuid.UUID) (storegen.GetEducationSystemRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(storegen.GetEducationSystemRow), args.Error(1)
}

func (m *mockEduStore) ListEducationSystemLevels(ctx context.Context, educationSystemID uuid.UUID) ([]storegen.ListEducationSystemLevelsRow, error) {
	args := m.Called(ctx, educationSystemID)
	return args.Get(0).([]storegen.ListEducationSystemLevelsRow), args.Error(1)
}

func (m *mockEduStore) ListEducationSystemGradingBands(ctx context.Context, educationSystemID uuid.UUID) ([]storegen.ListEducationSystemGradingBandsRow, error) {
	args := m.Called(ctx, educationSystemID)
	return args.Get(0).([]storegen.ListEducationSystemGradingBandsRow), args.Error(1)
}

func (m *mockEduStore) ListEducationSystemSubjects(ctx context.Context, educationSystemID uuid.UUID) ([]storegen.ListEducationSystemSubjectsRow, error) {
	args := m.Called(ctx, educationSystemID)
	return args.Get(0).([]storegen.ListEducationSystemSubjectsRow), args.Error(1)
}

func (m *mockEduStore) ListEducationSystemTermStructures(ctx context.Context, educationSystemID uuid.UUID) ([]storegen.EducationSystemTermStructure, error) {
	args := m.Called(ctx, educationSystemID)
	return args.Get(0).([]storegen.EducationSystemTermStructure), args.Error(1)
}

func newTestService(s store.StoreInterface) *Service {
	return &Service{
		store: s,
		cfg:   &config.Config{},
		log:   logger.New("info"),
	}
}

func TestService_ListCountries_Success(t *testing.T) {
	ctx := context.Background()

	countries := []storegen.Country{
		{ID: uuid.New(), Name: "Kenya", Code: "KE", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New(), Name: "Uganda", Code: "UG", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	mockStore := new(mockEduStore)
	mockStore.On("ListCountries", ctx).Return(countries, nil).Once()

	svc := newTestService(mockStore)

	resp, err := svc.ListCountries(ctx)

	require.NoError(t, err)
	assert.Len(t, resp.Countries, 2)
	assert.Equal(t, "Kenya", resp.Countries[0].Name)
	assert.Equal(t, "KE", resp.Countries[0].Code)
	assert.Equal(t, countries[0].ID.String(), resp.Countries[0].ID)
	mockStore.AssertExpectations(t)
}

func TestService_ListCountries_StoreError(t *testing.T) {
	ctx := context.Background()

	mockStore := new(mockEduStore)
	mockStore.On("ListCountries", ctx).Return([]storegen.Country{}, errors.New("db connection failed")).Once()

	svc := newTestService(mockStore)

	resp, err := svc.ListCountries(ctx)

	require.Error(t, err)
	assert.True(t, apperr.IsCode(err, apperr.INTERNAL_ERROR))
	assert.Empty(t, resp.Countries)
	mockStore.AssertExpectations(t)
}

func TestService_ListEducationSystems_Success(t *testing.T) {
	ctx := context.Background()

	systems := []storegen.ListEducationSystemsRow{
		{
			ID:                  uuid.New(),
			CountryID:           uuid.New(),
			Name:                "CBC",
			YearsPrimary:        6,
			YearsLowerSecondary: 3,
			YearsUpperSecondary: 3,
			DefaultLanguage:     "English",
			CountryName:         "Kenya",
			CountryCode:         "KE",
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		},
	}

	mockStore := new(mockEduStore)
	mockStore.On("ListEducationSystems", ctx).Return(systems, nil).Once()

	svc := newTestService(mockStore)

	resp, err := svc.ListEducationSystems(ctx)

	require.NoError(t, err)
	assert.Len(t, resp.Systems, 1)
	assert.Equal(t, "CBC", resp.Systems[0].Name)
	assert.Equal(t, "Kenya", resp.Systems[0].CountryName)
	assert.Equal(t, "KE", resp.Systems[0].CountryCode)
	assert.Equal(t, int32(6), resp.Systems[0].YearsPrimary)
	assert.Equal(t, int32(3), resp.Systems[0].YearsLowerSecondary)
	assert.Equal(t, int32(3), resp.Systems[0].YearsUpperSecondary)
	mockStore.AssertExpectations(t)
}

func TestService_ListEducationSystems_StoreError(t *testing.T) {
	ctx := context.Background()

	mockStore := new(mockEduStore)
	mockStore.On("ListEducationSystems", ctx).Return([]storegen.ListEducationSystemsRow{}, errors.New("query failed")).Once()

	svc := newTestService(mockStore)

	_, err := svc.ListEducationSystems(ctx)

	require.Error(t, err)
	assert.True(t, apperr.IsCode(err, apperr.INTERNAL_ERROR))
	mockStore.AssertExpectations(t)
}

func TestService_GetEducationSystem_Success(t *testing.T) {
	ctx := context.Background()
	systemID := uuid.New()

	system := storegen.GetEducationSystemRow{
		ID:                  systemID,
		CountryID:           uuid.New(),
		Name:                "CBC",
		YearsPrimary:        6,
		YearsLowerSecondary: 3,
		YearsUpperSecondary: 3,
		DefaultLanguage:     "English",
		CountryName:         "Kenya",
		CountryCode:         "KE",
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	levels := []storegen.ListEducationSystemLevelsRow{
		{
			EducationLevelID1: uuid.New(),
			LevelName:         "Primary",
			SortOrder:         1,
			YearCount:         6,
			LevelCreatedAt:    time.Now(),
		},
		{
			EducationLevelID1: uuid.New(),
			LevelName:         "Lower Secondary",
			SortOrder:         2,
			YearCount:         3,
			LevelCreatedAt:    time.Now(),
		},
	}

	bands := []storegen.ListEducationSystemGradingBandsRow{
		{
			ID:             uuid.New(),
			Grade:          "EE1",
			ScaleType:      "rubric",
			RubricLevel:    sqlNullString("EE1"),
			MinScore:       "80",
			MaxScore:       "100",
			SortOrder:      1,
			ScaleName:      "CBC Scale",
			IsDefault:      true,
			CreatedAt:      time.Now(),
		},
		{
			ID:             uuid.New(),
			Grade:          "ME1",
			ScaleType:      "rubric",
			RubricLevel:    sqlNullString("ME1"),
			MinScore:       "65",
			MaxScore:       "79",
			SortOrder:      2,
			ScaleName:      "CBC Scale",
			IsDefault:      true,
			CreatedAt:      time.Now(),
		},
	}

	subjects := []storegen.ListEducationSystemSubjectsRow{
		{
			ID:                  uuid.New(),
			SubjectCode:         "MATH",
			SubjectName:         "Mathematics",
			IsCompulsoryDefault: true,
			LevelName:           "Primary",
			SortOrder:           1,
			CreatedAt:           time.Now(),
		},
		{
			ID:                  uuid.New(),
			SubjectCode:         "ENG",
			SubjectName:         "English",
			IsCompulsoryDefault: true,
			LevelName:           "Primary",
			SortOrder:           1,
			CreatedAt:           time.Now(),
		},
	}

	terms := []storegen.EducationSystemTermStructure{
		{ID: uuid.New(), EducationSystemID: systemID, TermName: "Term 1", TermSortOrder: 1, CreatedAt: time.Now()},
		{ID: uuid.New(), EducationSystemID: systemID, TermName: "Term 2", TermSortOrder: 2, CreatedAt: time.Now()},
		{ID: uuid.New(), EducationSystemID: systemID, TermName: "Term 3", TermSortOrder: 3, CreatedAt: time.Now()},
	}

	mockStore := new(mockEduStore)
	mockStore.On("GetEducationSystem", ctx, systemID).Return(system, nil).Once()
	mockStore.On("ListEducationSystemLevels", ctx, systemID).Return(levels, nil).Once()
	mockStore.On("ListEducationSystemGradingBands", ctx, systemID).Return(bands, nil).Once()
	mockStore.On("ListEducationSystemSubjects", ctx, systemID).Return(subjects, nil).Once()
	mockStore.On("ListEducationSystemTermStructures", ctx, systemID).Return(terms, nil).Once()

	svc := newTestService(mockStore)

	resp, err := svc.GetEducationSystem(ctx, systemID.String())

	require.NoError(t, err)
	assert.Equal(t, systemID.String(), resp.ID)
	assert.Equal(t, "CBC", resp.Name)
	assert.Equal(t, "Kenya", resp.CountryName)
	assert.Equal(t, "KE", resp.CountryCode)
	assert.Equal(t, "English", resp.DefaultLanguage)

	// Verify levels
	assert.Len(t, resp.Levels, 2)
	assert.Equal(t, "Primary", resp.Levels[0].LevelName)
	assert.Equal(t, int32(6), resp.Levels[0].YearCount)

	// Verify grading bands
	assert.Len(t, resp.GradingBands, 2)
	assert.Equal(t, "EE1", resp.GradingBands[0].Grade)
	assert.Equal(t, "rubric", resp.GradingBands[0].ScaleType)
	assert.NotNil(t, resp.GradingBands[0].RubricLevel)
	assert.Equal(t, "EE1", *resp.GradingBands[0].RubricLevel)

	// Verify subjects
	assert.Len(t, resp.Subjects, 2)
	assert.Equal(t, "MATH", resp.Subjects[0].SubjectCode)
	assert.True(t, resp.Subjects[0].IsCompulsoryDefault)

	// Verify term structures
	assert.Len(t, resp.TermStructures, 3)
	assert.Equal(t, "Term 1", resp.TermStructures[0].TermName)
	assert.Equal(t, int32(1), resp.TermStructures[0].TermSortOrder)

	mockStore.AssertExpectations(t)
}

func TestService_GetEducationSystem_InvalidID(t *testing.T) {
	ctx := context.Background()

	mockStore := new(mockEduStore)
	svc := newTestService(mockStore)

	resp, err := svc.GetEducationSystem(ctx, "not-a-uuid")

	require.Error(t, err)
	assert.True(t, apperr.IsCode(err, apperr.INVALID_REQUEST))
	assert.Empty(t, resp.ID)
	mockStore.AssertNotCalled(t, "GetEducationSystem", mock.Anything, mock.Anything)
}

func TestService_GetEducationSystem_NotFound(t *testing.T) {
	ctx := context.Background()
	systemID := uuid.New()

	mockStore := new(mockEduStore)
	mockStore.On("GetEducationSystem", ctx, systemID).Return(storegen.GetEducationSystemRow{}, errors.New("sql: no rows")).Once()

	svc := newTestService(mockStore)

	resp, err := svc.GetEducationSystem(ctx, systemID.String())

	require.Error(t, err)
	assert.True(t, apperr.IsCode(err, apperr.NOT_FOUND))
	assert.Empty(t, resp.ID)
	mockStore.AssertExpectations(t)
}

func TestService_GetEducationSystem_StoreError(t *testing.T) {
	ctx := context.Background()
	systemID := uuid.New()

	mockStore := new(mockEduStore)
	mockStore.On("GetEducationSystem", ctx, systemID).Return(storegen.GetEducationSystemRow{}, errors.New("db error")).Once()

	svc := newTestService(mockStore)

	_, err := svc.GetEducationSystem(ctx, systemID.String())

	require.Error(t, err)
	assert.True(t, apperr.IsCode(err, apperr.NOT_FOUND))
	mockStore.AssertExpectations(t)
}

// sqlNullString is a helper to create sql.NullString.
func sqlNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}
