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
			return WriteError(c, ErrBadRequest("id required", nil))
		}

		detail, err := curriculumSvc.GetSubjectDetail(ctx, id)
		if err != nil {
			if strings.Contains(err.Error(), "bad_request:") {
				return WriteError(c, ErrBadRequest(err.Error(), nil))
			}
			if strings.Contains(err.Error(), "not_found") || strings.Contains(err.Error(), "internal_error") {
				return WriteError(c, ErrNotFound("subject not found"))
			}
			return WriteError(c, ErrInternal(err.Error()))
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
