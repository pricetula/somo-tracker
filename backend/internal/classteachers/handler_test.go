package classteachers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

type handlerTestHarness struct {
	app  *fiber.App
	svc  *Service
	repo *MockRepository
}

func newHandlerTestHarness(t *testing.T) *handlerTestHarness {
	t.Helper()
	repo := &MockRepository{}
	svc := NewService(repo)
	handler := NewHandler(svc)

	app := fiber.New()

	testAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		c.Locals("school_id", "school_001")
		c.Locals("role", "SCHOOL_ADMIN")
		return c.Next()
	}

	ct := app.Group("/api/v1/class-teachers", testAuth)
	ct.Post("/", handler.Create)
	ct.Get("/by-class/:classId", handler.ListByClass)
	ct.Get("/by-teacher/:userId", handler.ListByTeacher)
	ct.Get("/:id", handler.GetByID)
	ct.Delete("/", handler.Delete)

	return &handlerTestHarness{app: app, svc: svc, repo: repo}
}

func doRequest(app *fiber.App, method, path string, body []byte) *http.Response {
	req := httptest.NewRequest(method, path, nil)
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	resp, _ := app.Test(req)
	return resp
}

func TestHandler_CreatePrimaryTeacher(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.countPrimaryFn = func(ctx context.Context, classID, tenantID string) (int, error) {
		return 0, nil
	}
	h.repo.createFn = func(ctx context.Context, params CreateClassTeacherParams) (string, error) {
		return "ct_001", nil
	}
	h.repo.getByIDFn = func(ctx context.Context, id, tenantID string) (*ClassTeacher, error) {
		return &ClassTeacher{
			ID: id, ClassID: "class_001", UserID: "user_001",
			TeacherRole: "PRIMARY_CLASS_TEACHER",
		}, nil
	}

	body, _ := json.Marshal(CreateClassTeacherPayload{
		UserID:      "user_001",
		ClassID:     "class_001",
		TeacherRole: "PRIMARY_CLASS_TEACHER",
	})
	resp := doRequest(h.app, "POST", "/api/v1/class-teachers", body)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result ClassTeacher
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.Equal(t, "PRIMARY_CLASS_TEACHER", result.TeacherRole)
}

func TestHandler_CreatePrimaryAlreadyAssigned(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.countPrimaryFn = func(ctx context.Context, classID, tenantID string) (int, error) {
		return 1, nil
	}

	body, _ := json.Marshal(CreateClassTeacherPayload{
		UserID:      "user_001",
		ClassID:     "class_001",
		TeacherRole: "PRIMARY_CLASS_TEACHER",
	})
	resp := doRequest(h.app, "POST", "/api/v1/class-teachers", body)
	require.Equal(t, fiber.StatusConflict, resp.StatusCode)

	var errResp errorResponse
	err := json.NewDecoder(resp.Body).Decode(&errResp)
	require.NoError(t, err)
	require.Equal(t, "primary_already_assigned", errResp.Code)
}

func TestHandler_CreateInvalidInput(t *testing.T) {
	h := newHandlerTestHarness(t)

	// Missing user_id
	body, _ := json.Marshal(CreateClassTeacherPayload{
		ClassID:     "class_001",
		TeacherRole: "SUBJECT_TEACHER",
	})
	resp := doRequest(h.app, "POST", "/api/v1/class-teachers", body)
	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestHandler_GetByID(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID string) (*ClassTeacher, error) {
		return &ClassTeacher{ID: id, UserID: "user_001", TeacherRole: "SUBJECT_TEACHER"}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/class-teachers/ct_001", nil)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result ClassTeacher
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.Equal(t, "ct_001", result.ID)
}

func TestHandler_GetByID_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID string) (*ClassTeacher, error) {
		return nil, ErrNotFound
	}

	resp := doRequest(h.app, "GET", "/api/v1/class-teachers/ct_missing", nil)
	require.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestHandler_ListByClass(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listByClassFn = func(ctx context.Context, classID, tenantID string) ([]ClassTeacher, error) {
		return []ClassTeacher{
			{ID: "ct_001", UserID: "user_001", TeacherRole: "PRIMARY_CLASS_TEACHER"},
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/class-teachers/by-class/class_001", nil)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result ClassTeacherListResponse
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
}

func TestHandler_ListByTeacher(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listByTeacherFn = func(ctx context.Context, userID, tenantID string) ([]ClassTeacher, error) {
		return []ClassTeacher{
			{ID: "ct_001", ClassID: "class_001", TeacherRole: "SUBJECT_TEACHER"},
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/class-teachers/by-teacher/user_001", nil)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result ClassTeacherListResponse
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
}

func TestHandler_Delete(t *testing.T) {
	h := newHandlerTestHarness(t)

	var deletedID string
	h.repo.deleteFn = func(ctx context.Context, id, tenantID string) error {
		deletedID = id
		return nil
	}

	body, _ := json.Marshal(struct {
		ID string `json:"id"`
	}{ID: "ct_001"})
	resp := doRequest(h.app, "DELETE", "/api/v1/class-teachers", body)
	require.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	require.Equal(t, "ct_001", deletedID)
}
