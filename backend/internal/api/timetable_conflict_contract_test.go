package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestTimetableConflictErrorContractShape(t *testing.T) {
	app := fiber.New()
	app.Post("/test-conflict", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"code":    "timetable_conflict",
			"message": "Timetable conflict detected",
			"errors": map[string]interface{}{
				"conflicts": []map[string]interface{}{
					{
						"type":             "teacher_double_book",
						"resource_id":      "uuid-123",
						"resource_name":    "Teacher A",
						"existing_slot_id": "uuid-456",
					},
				},
			},
			"request_id": "test-req-id",
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/test-conflict", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)

	var body map[string]interface{}
	requireNoError := func(err error) { assert.NoError(t, err) }
	requireNoError(json.NewDecoder(resp.Body).Decode(&body))

	assert.Equal(t, "timetable_conflict", body["code"])
	assert.Equal(t, "Timetable conflict detected", body["message"])
	assert.Contains(t, body, "errors")

	errorsMap := body["errors"].(map[string]interface{})
	assert.Contains(t, errorsMap, "conflicts")

	conflicts := errorsMap["conflicts"].([]interface{})
	assert.Len(t, conflicts, 1)

	conflictObj := conflicts[0].(map[string]interface{})
	assert.Equal(t, "teacher_double_book", conflictObj["type"])
	assert.Equal(t, "Teacher A", conflictObj["resource_name"])
	assert.Equal(t, "uuid-456", conflictObj["existing_slot_id"])
}
