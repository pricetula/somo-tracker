package api

import (
	"strconv"

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
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "topic_id or id required", "errors": fiber.Map{}})
			}
			sub, err := curriculumSvc.GetSubTopic(ctx, id)
			if err != nil {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "sub topic not found", "errors": fiber.Map{}})
			}
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"id":      sub.ID,
				"topicId": sub.TopicID,
				"name":    sub.Name,
				"code":    sub.Code,
			})
		}

		page, _ := strconv.Atoi(c.Query("page", "1"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(c.Query("limit", "50"))
		if limit < 1 || limit > 200 {
			limit = 50
		}

		items, total, err := curriculumSvc.ListSubTopics(ctx, topicID, page, limit)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": err.Error(), "errors": fiber.Map{}})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"code":   "sub_topics_fetched",
			"items":  items,
			"total":  total,
			"errors": fiber.Map{},
		})
	}
}
