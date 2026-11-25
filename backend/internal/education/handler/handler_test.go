package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	storegen "somotracker/internal/auth/store/gen"
	"somotracker/internal/platform/config"
	"somotracker/internal/education/service"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/logger"
)

// mockEduStore implements store.StoreInterface using testify mock.
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

func newTestHandlerWithStore(mockStore *mockEduStore) *Handler {
	svc := service.New(mockStore, &config.Config{}, logger.New("info"))
	return &Handler{
		svc:  svc,
		cfg:  &config.Config{},
		log:  logger.New("info"),
	}
}

func TestHandler_ListCountries_Success(t *testing.T) {
	app := fiber.New()

	countries := []storegen.Country{
		{ID: uuid.New(), Name: "Kenya", Code: "KE", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New(), Name: "Uganda", Code: "UG", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	mockStore := new(mockEduStore)
	mockStore.On("ListCountries", mock.Anything).Return(countries, nil).Once()

	h := newTestHandlerWithStore(mockStore)
	app.Get("/api/v1/education/countries", h.ListCountries)

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/education/countries", nil))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := strings.TrimSpace(string(bodyBytes))
	assert.Contains(t, body, "Kenya")
	assert.Contains(t, body, "KE")
	assert.Contains(t, body, "Uganda")
	assert.Contains(t, body, "UG")
	mockStore.AssertExpectations(t)
}

func TestHandler_ListCountries_ServiceError(t *testing.T) {
	app := fiber.New()

	mockStore := new(mockEduStore)
	mockStore.On("ListCountries", mock.Anything).Return([]storegen.Country{}, assert.AnError).Once()

	h := newTestHandlerWithStore(mockStore)
	app.Get("/api/v1/education/countries", h.ListCountries)

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/education/countries", nil))
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	mockStore.AssertExpectations(t)
}

func TestHandler_ListEducationSystems_Success(t *testing.T) {
	app := fiber.New()

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
	mockStore.On("ListEducationSystems", mock.Anything).Return(systems, nil).Once()

	h := newTestHandlerWithStore(mockStore)
	app.Get("/api/v1/education/systems", h.ListEducationSystems)

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/education/systems", nil))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := strings.TrimSpace(string(bodyBytes))
	assert.Contains(t, body, "CBC")
	assert.Contains(t, body, "Kenya")
	assert.Contains(t, body, "KE")
	mockStore.AssertExpectations(t)
}

func TestHandler_GetEducationSystem_Success(t *testing.T) {
	app := fiber.New()

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

	mockStore := new(mockEduStore)
	mockStore.On("GetEducationSystem", mock.Anything, systemID).Return(system, nil).Once()
	mockStore.On("ListEducationSystemLevels", mock.Anything, systemID).Return([]storegen.ListEducationSystemLevelsRow{}, nil).Once()
	mockStore.On("ListEducationSystemGradingBands", mock.Anything, systemID).Return([]storegen.ListEducationSystemGradingBandsRow{}, nil).Once()
	mockStore.On("ListEducationSystemSubjects", mock.Anything, systemID).Return([]storegen.ListEducationSystemSubjectsRow{}, nil).Once()
	mockStore.On("ListEducationSystemTermStructures", mock.Anything, systemID).Return([]storegen.EducationSystemTermStructure{}, nil).Once()

	h := newTestHandlerWithStore(mockStore)
	app.Get("/api/v1/education/systems/:id", h.GetEducationSystem)

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/education/systems/"+systemID.String(), nil))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := strings.TrimSpace(string(bodyBytes))
	assert.Contains(t, body, systemID.String())
	assert.Contains(t, body, "CBC")
	assert.Contains(t, body, "Kenya")
	assert.Contains(t, body, "KE")
	mockStore.AssertExpectations(t)
}

func TestHandler_GetEducationSystem_NotFound(t *testing.T) {
	app := fiber.New()

	systemID := uuid.New()
	mockStore := new(mockEduStore)
	mockStore.On("GetEducationSystem", mock.Anything, systemID).Return(storegen.GetEducationSystemRow{}, apperr.New(apperr.NOT_FOUND, "not found")).Once()

	h := newTestHandlerWithStore(mockStore)
	app.Get("/api/v1/education/systems/:id", h.GetEducationSystem)

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/education/systems/"+systemID.String(), nil))
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	mockStore.AssertExpectations(t)
}

func TestHandler_ListEducationSystems_ServiceError(t *testing.T) {
	app := fiber.New()

	mockStore := new(mockEduStore)
	mockStore.On("ListEducationSystems", mock.Anything).Return([]storegen.ListEducationSystemsRow{}, assert.AnError).Once()

	h := newTestHandlerWithStore(mockStore)
	app.Get("/api/v1/education/systems", h.ListEducationSystems)

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/education/systems", nil))
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	mockStore.AssertExpectations(t)
}
