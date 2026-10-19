package api

import (
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

		pagination, err := ValidatePagination(c, 100)
		if err != nil {
			return err
		}
		page, limit := pagination.Page, pagination.Limit

		items, total, err := curriculumSvc.ListTopics(ctx, subjectID, page, limit)
		if err != nil {
			return WriteError(c, ErrInternal("Something went wrong"))
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"code":   "topics_fetched",
			"items":  items,
			"total":  total,
			"page":   page,
			"limit":  limit,
			"errors": fiber.Map{},
		})
	}
}
