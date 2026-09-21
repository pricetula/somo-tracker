# E2E Handler Test Template

Use this template to scaffold HTTP handler E2E tests for the backend test pyramid.

## File: internal/api/<handler>_handler_test.go

```go
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"somotracker/backend/internal/services"
)

type mock<Handler>Service struct {
	// add function fields matching the service interface
	listFunc func(ctx context.Context, schoolID uuid.UUID, ...) (..., error)
}

func (m *mock<Handler>Service) <MethodName>(ctx context.Context, ...) (...) {
	if m.listFunc != nil { return m.listFunc(ctx, ...) }
	return ..., nil
}

func new<Handler>TestApp(svc services.<Handler>Service, schoolID, userID string) *fiber.App {
	app := fiber.New()
	h := New<Handler>Handler(svc, nil)
	app.Get("/path", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID)
		c.Locals("user_id", userID)
		return h.<Method>(c)
	})
	return app
}

func Test<Handler>Handler_HappyPath(t *testing.T) {
	t.Parallel()
	schoolID := uuid.New()
	svc := &mock<Handler>Service{
		listFunc: func(ctx context.Context, schoolID uuid.UUID, ...) (...) {
			return ..., nil
		},
	}
	app := new<Handler>TestApp(svc, schoolID.String(), uuid.New().String())
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/path", nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
```

## Rules

1. Mock only the service interface, no DB.
2. Inject `active_school_id`, `tenant_id`, `user_id` via `c.Locals`.
3. Use `app.Test` for full router + middleware path.
4. Assert canonical error JSON for non-2xx.
5. Keep tests fast, parallelizable, no Docker.

Copy this template for each handler and fill in the real service interface signatures.
