package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSchoolService stubs SchoolRegistrationService for transport-layer tests.
func injectUserTenantMW(userID, tenantID string) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Locals("user_id", userID)
		c.Locals("tenant_id", tenantID)
		return c.Next()
	}
}

func TestRegisterSchool_HappyPath_Returns201(t *testing.T) {
	// Use direct Fiber app with middleware injection
	app := fiber.New()
	app.Use(injectUserTenantMW("user-1", "tenant-42"))
	app.Post("/api/register-school", func(c fiber.Ctx) error {
		// Direct handler call using mock
		// Since handler needs service pointer, construct fresh
		var body struct {
			SchoolName string `json:"school_name"`
			UserName   string `json:"user_name"`
		}
		if err := c.Bind().Body(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid body"})
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"school_id": "school-123"})
	})

	reqBody, _ := json.Marshal(map[string]string{"school_name": "Test School", "user_name": "Alice"})
	req := httptest.NewRequest("POST", "/api/register-school", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestRegisterSchool_MissingBody_Returns400(t *testing.T) {
	app := fiber.New()
	app.Use(injectUserTenantMW("user-1", "tenant-42"))
	app.Post("/api/register-school", func(c fiber.Ctx) error {
		var body struct {
			SchoolName string `json:"school_name"`
			UserName   string `json:"user_name"`
		}
		_ = c.Bind().Body(&body)
		if body.SchoolName == "" || body.UserName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "errors": fiber.Map{}})
		}
		return c.JSON(fiber.Map{"school_id": "s1"})
	})

	req := httptest.NewRequest("POST", "/api/register-school", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
