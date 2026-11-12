package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/authz"
	"somotracker/internal/students/service"
)

// fakeService implements the service interface for handler tests.
type fakeService struct {
	students      map[uuid.UUID]*service.StudentDTO
	guardianLinks map[uuid.UUID][]service.GuardianLinkDTO
	err           error
}

func newFakeService() *fakeService {
	return &fakeService{
		students:      make(map[uuid.UUID]*service.StudentDTO),
		guardianLinks: make(map[uuid.UUID][]service.GuardianLinkDTO),
	}
}

func (f *fakeService) ListStudents(ctx context.Context, guardianID *uuid.UUID, limit int32, cursorFullName string, cursorStudentID uuid.UUID) (*service.ListResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	items := make([]service.StudentDTO, 0, len(f.students))
	for _, s := range f.students {
		items = append(items, *s)
	}
	return &service.ListResponse{Items: items}, nil
}

func (f *fakeService) GetStudent(ctx context.Context, id uuid.UUID) (*service.StudentDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	s, ok := f.students[id]
	if !ok {
		return nil, &apperr.Error{Code: apperr.NOT_FOUND, Message: "record not found"}
	}
	return s, nil
}

func (f *fakeService) CreateStudent(ctx context.Context, req service.CreateStudentRequest) (*service.StudentDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	id := uuid.New()
	dto := &service.StudentDTO{
		StudentID:       id,
		SchoolID:        req.SchoolID,
		AdmissionNumber: req.AdmissionNumber,
		FullName:        req.FullName,
		DateOfBirth:     req.DateOfBirth,
		Gender:          req.Gender,
		Metadata:        req.Metadata,
	}
	f.students[id] = dto
	return dto, nil
}

func (f *fakeService) UpdateStudent(ctx context.Context, req service.UpdateStudentRequest) (*service.StudentDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	s, ok := f.students[req.StudentID]
	if !ok {
		return nil, &apperr.Error{Code: apperr.NOT_FOUND, Message: "record not found"}
	}
	s.AdmissionNumber = req.AdmissionNumber
	s.FullName = req.FullName
	s.DateOfBirth = req.DateOfBirth
	s.Gender = req.Gender
	s.Metadata = req.Metadata
	return s, nil
}

func (f *fakeService) DeleteStudent(ctx context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.students[id]; !ok {
		return &apperr.Error{Code: apperr.NOT_FOUND, Message: "record not found"}
	}
	delete(f.students, id)
	return nil
}

func (f *fakeService) ListGuardianLinks(ctx context.Context, studentID uuid.UUID) ([]service.GuardianLinkDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.guardianLinks[studentID], nil
}

func (f *fakeService) CreateGuardianLink(ctx context.Context, req service.CreateGuardianLinkRequest) (*service.GuardianLinkDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	dto := &service.GuardianLinkDTO{
		GuardianID:         req.GuardianID,
		StudentID:          req.StudentID,
		RelationshipType:   req.RelationshipType,
		IsEmergencyContact: req.IsEmergencyContact,
	}
	f.guardianLinks[req.StudentID] = append(f.guardianLinks[req.StudentID], *dto)
	return dto, nil
}

func (f *fakeService) DeleteGuardianLink(ctx context.Context, studentID uuid.UUID, guardianID uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	links := f.guardianLinks[studentID]
	for i, link := range links {
		if link.GuardianID == guardianID {
			f.guardianLinks[studentID] = append(links[:i], links[i+1:]...)
			return nil
		}
	}
	return &apperr.Error{Code: apperr.NOT_FOUND, Message: "record not found"}
}

// requirePermissionMiddleware checks if any of the roles has the required permission.
// Simulates authz.Require for testing.
func requirePermissionMiddleware(perm authz.Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		rolesRaw, ok := c.Locals("roles").([]string)
		if !ok || len(rolesRaw) == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "UNAUTHENTICATED", "message": "authentication required"})
		}
		if !authz.HasPermission(toRoles(rolesRaw), perm) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"code": "FORBIDDEN", "message": "insufficient permission"})
		}
		return c.Next()
	}
}

func toRoles(roles []string) []authz.Role {
	result := make([]authz.Role, len(roles))
	for i, r := range roles {
		result[i] = authz.Role(r)
	}
	return result
}

// buildTestApp creates a Fiber app with the handler routes and permission middleware.
// Roles are set via c.Locals("roles") by a middleware before the handler.
func buildTestApp(t *testing.T, h *Handler, roles []string) *fiber.App {
	t.Helper()
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := http.StatusInternalServerError
			msg := err.Error()
			ecode := "INTERNAL_ERROR"
			var appErr *apperr.Error
			if errors.As(err, &appErr) {
				code = apperr.Status(appErr.Code)
				if code == 0 {
					code = http.StatusInternalServerError
				}
				msg = appErr.Message
				ecode = string(appErr.Code)
			}
			return c.Status(code).JSON(fiber.Map{"code": ecode, "message": msg})
		},
	})

	// Middleware: set roles and school IDs from session (simulates AuthMiddleware)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("roles", roles)
		c.Locals("school_ids", []uuid.UUID{uuid.New()})
		c.Locals("guardian_id", uuid.New()) // simulate guardian membership in session
		return c.Next()
	})

	// Students routes with permission middleware
	students := app.Group("/api/v1/students")

	students.Get("/", requirePermissionMiddleware(authz.PermStudentRead), h.ListStudents)
	students.Get("/:id", requirePermissionMiddleware(authz.PermStudentRead), h.GetStudent)
	students.Post("/", requirePermissionMiddleware(authz.PermStudentWrite), h.CreateStudent)
	students.Put("/:id", requirePermissionMiddleware(authz.PermStudentWrite), h.UpdateStudent)
	students.Delete("/:id", requirePermissionMiddleware(authz.PermStudentWrite), h.DeleteStudent)
	students.Get("/:id/guardians", requirePermissionMiddleware(authz.PermStudentRead), h.ListGuardianLinks)
	students.Post("/:id/guardians", requirePermissionMiddleware(authz.PermStudentWrite), h.CreateGuardianLink)
	students.Delete("/:id/guardians/:guardian_id", requirePermissionMiddleware(authz.PermStudentWrite), h.DeleteGuardianLink)

	return app
}

// TestHandler_ListStudents_Permission tests that only roles with student:read can list students.
func TestHandler_ListStudents_Permission(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	rolesWithRead := []string{"owner", "admin", "headteacher", "teacher", "guardian", "platformoperator"}
	rolesWithoutRead := []string{"finance"}

	for _, role := range rolesWithRead {
		t.Run(role+"_can_read", func(t *testing.T) {
			app := buildTestApp(t, h, []string{role})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/students", nil)
			resp, err := app.Test(req, fiber.TestConfig{})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}

	for _, role := range rolesWithoutRead {
		t.Run(role+"_cannot_read", func(t *testing.T) {
			app := buildTestApp(t, h, []string{role})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/students", nil)
			resp, err := app.Test(req, fiber.TestConfig{})
			require.NoError(t, err)
			require.Equal(t, http.StatusForbidden, resp.StatusCode)
		})
	}
}

// TestHandler_CreateStudent_Permission tests that only roles with student:write can create students.
func TestHandler_CreateStudent_Permission(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	rolesWithWrite := []string{"owner", "admin", "headteacher", "platformoperator"}
	rolesWithoutWrite := []string{"teacher", "guardian", "finance"}

	body := `{"admission_number":"A001","full_name":"Alice Johnson","date_of_birth":"2010-01-15","gender":"F","metadata":{}}`

	for _, role := range rolesWithWrite {
		t.Run(role+"_can_write", func(t *testing.T) {
			app := buildTestApp(t, h, []string{role})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/students", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, fiber.TestConfig{})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode)
		})
	}

	for _, role := range rolesWithoutWrite {
		t.Run(role+"_cannot_write", func(t *testing.T) {
			app := buildTestApp(t, h, []string{role})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/students", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, fiber.TestConfig{})
			require.NoError(t, err)
			require.Equal(t, http.StatusForbidden, resp.StatusCode)
		})
	}
}

// TestHandler_GetStudent_NotFound tests that GetStudent returns 404 for missing student.
func TestHandler_GetStudent_NotFound(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	app := buildTestApp(t, h, []string{"teacher"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/students/"+uuid.New().String(), nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestHandler_CreateStudent_InvalidBody tests that invalid body returns 400.
func TestHandler_CreateStudent_InvalidBody(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	app := buildTestApp(t, h, []string{"headteacher"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/students", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestHandler_CreateStudent_MissingFields tests that missing required fields returns 422.
func TestHandler_CreateStudent_MissingFields(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	app := buildTestApp(t, h, []string{"headteacher"})
	body := `{"full_name":"Alice"}` // missing admission_number, date_of_birth, gender
	req := httptest.NewRequest(http.MethodPost, "/api/v1/students", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

// TestHandler_UpdateStudent_NotFound tests that UpdateStudent returns 404 for missing student.
func TestHandler_UpdateStudent_NotFound(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	app := buildTestApp(t, h, []string{"headteacher"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/students/"+uuid.New().String(), strings.NewReader(`{"admission_number":"A001","full_name":"Alice","date_of_birth":"2010-01-15","gender":"F"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestHandler_DeleteStudent_NotFound tests that DeleteStudent returns 404 for missing student.
func TestHandler_DeleteStudent_NotFound(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	app := buildTestApp(t, h, []string{"headteacher"})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/students/"+uuid.New().String(), nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestHandler_CreateGuardianLink_Permission tests guardian link write permission.
func TestHandler_CreateGuardianLink_Permission(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	studentID := uuid.New()
	body := `{"relationship_type":"Parent","is_emergency_contact":true}`

	// Headteacher can write guardian links (student:write)
	app := buildTestApp(t, h, []string{"headteacher"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/students/"+studentID.String()+"/guardians", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

// TestHandler_ListGuardianLinks_Permission tests guardian links read permission.
func TestHandler_ListGuardianLinks_Permission(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	studentID := uuid.New()

	// Teacher can read guardian links (student:read)
	app := buildTestApp(t, h, []string{"teacher"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/students/"+studentID.String()+"/guardians", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestHandler_DeleteGuardianLink_Permission tests guardian link delete permission.
func TestHandler_DeleteGuardianLink_Permission(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	studentID := uuid.New()
	guardianID := uuid.New()

	// Create a link first so it exists for deletion
	_, err := svc.CreateGuardianLink(context.Background(), service.CreateGuardianLinkRequest{
		GuardianID:         guardianID,
		StudentID:          studentID,
		RelationshipType:   "Parent",
		IsEmergencyContact: true,
	})
	require.NoError(t, err)

	// Headteacher can delete guardian links (student:write)
	app := buildTestApp(t, h, []string{"headteacher"})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/students/"+studentID.String()+"/guardians/"+guardianID.String(), nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

// TestHandler_ErrorResponseShape tests that error responses have the correct shape.
func TestHandler_ErrorResponseShape(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	app := buildTestApp(t, h, []string{"teacher"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/students/"+uuid.New().String(), nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var respMap map[string]interface{}
	require.NoError(t, json.Unmarshal(bodyBytes, &respMap))
	require.Contains(t, respMap, "code")
	require.Contains(t, respMap, "message")
}
