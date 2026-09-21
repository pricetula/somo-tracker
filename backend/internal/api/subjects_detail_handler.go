package api

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"somotracker/backend/internal/services"
)

func subjectsDetailHandler(curriculumSvc services.CurriculumService) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := c.Context()
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "id required", "errors": fiber.Map{}})
		}

		detail, err := curriculumSvc.GetSubjectDetail(ctx, id)
		if err != nil {
			if strings.Contains(err.Error(), "bad_request:") {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": err.Error(), "errors": fiber.Map{}})
			}
			if strings.Contains(err.Error(), "not_found") || strings.Contains(err.Error(), "internal_error") {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "subject not found", "errors": fiber.Map{}})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": err.Error(), "errors": fiber.Map{}})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"id":     detail.ID,
			"name":   detail.Name,
			"code":   detail.Code,
			"color":  detail.Color,
			"grade":  detail.Grade,
			"topics": detail.Topics,
		})
	}
}
