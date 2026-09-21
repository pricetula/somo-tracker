package api

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"somotracker/backend/internal/services"
)

func topicsDetailHandler(curriculumSvc services.CurriculumService) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := c.Context()
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "id required", "errors": fiber.Map{}})
		}

		detail, err := curriculumSvc.GetTopicDetail(ctx, id)
		if err != nil {
			if strings.Contains(err.Error(), "bad_request:") {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": err.Error(), "errors": fiber.Map{}})
			}
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "topic not found", "errors": fiber.Map{}})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"id":          detail.ID,
			"subjectId":   detail.SubjectID,
			"name":        detail.Name,
			"code":        detail.Code,
			"description": detail.Description,
			"subTopics":   detail.SubTopics,
		})
	}
}
