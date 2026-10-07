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
			return WriteError(c, ErrBadRequest("id required", nil))
		}

		detail, err := curriculumSvc.GetTopicDetail(ctx, id)
		if err != nil {
			if strings.Contains(err.Error(), "bad_request:") {
				return WriteError(c, ErrBadRequest(err.Error(), nil))
			}
			return WriteError(c, ErrNotFound("topic not found"))
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
