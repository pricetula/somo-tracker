package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"somotracker/backend/internal/services"
)

func topicsListHandler(curriculumSvc services.CurriculumService) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := c.Context()
		subjectID := c.Query("subject_id")
		if subjectID == "" {
			return WriteError(c, ErrBadRequest("subject_id is required", nil))
		}
		page, _ := strconv.Atoi(c.Query("page", "1"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(c.Query("limit", "50"))
		if limit < 1 || limit > 200 {
			limit = 50
		}

		items, total, err := curriculumSvc.ListTopics(ctx, subjectID, page, limit)
		if err != nil {
			return WriteError(c, ErrInternal("Something went wrong"))
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"code":   "topics_fetched",
			"items":  items,
			"total":  total,
			"errors": fiber.Map{},
		})
	}
}
