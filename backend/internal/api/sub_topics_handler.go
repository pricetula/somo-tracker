package api

import (
	"github.com/gofiber/fiber/v3"

	"somotracker/backend/internal/services"
)

func subTopicsListHandler(curriculumSvc services.CurriculumService) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := c.Context()
		topicID := c.Query("topic_id")
		if topicID == "" {
			id := c.Params("id")
			if id == "" {
				return WriteError(c, ErrBadRequest("topic_id or id required", nil))
			}
			sub, err := curriculumSvc.GetSubTopic(ctx, id)
			if err != nil {
				return WriteError(c, ErrNotFound("sub topic not found"))
			}
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"id":      sub.ID,
				"topicId": sub.TopicID,
				"name":    sub.Name,
				"code":    sub.Code,
			})
		}

		pagination, err := ValidatePagination(c, 100)
		if err != nil {
			return err
		}
		page, limit := pagination.Page, pagination.Limit

		items, total, err := curriculumSvc.ListSubTopics(ctx, topicID, page, limit)
		if err != nil {
			return WriteError(c, ErrInternal("Something went wrong"))
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"code":   "sub_topics_fetched",
			"items":  items,
			"total":  total,
			"page":   page,
			"limit":  limit,
			"errors": fiber.Map{},
		})
	}
}
