package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

func subTopicsListHandler(pool *pgxpool.Pool) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := c.Context()
		topicID := c.Query("topic_id")
		if topicID == "" {
			// Single item fetch by id
			id := c.Params("id")
			if id == "" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "topic_id or id required", "errors": fiber.Map{}})
			}
			query := `SELECT id, topic_id, name, sequence_index FROM sub_topics WHERE id = $1`
			var idStr, topicIDStr, name string
			var seq int
			if err := pool.QueryRow(ctx, query, id).Scan(&idStr, &topicIDStr, &name, &seq); err != nil {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "sub topic not found", "errors": fiber.Map{}})
			}
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"id":      idStr,
				"topicId": topicIDStr,
				"name":    name,
				"code":    "",
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
		offset := (page - 1) * limit

		countSQL := `SELECT COUNT(*) FROM sub_topics WHERE topic_id = $1`
		var total int
		if err := pool.QueryRow(ctx, countSQL, topicID).Scan(&total); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to count sub topics", "errors": fiber.Map{}})
		}

		query := `SELECT id, topic_id, name, sequence_index FROM sub_topics WHERE topic_id = $1 ORDER BY sequence_index ASC LIMIT $2 OFFSET $3`
		rows, err := pool.Query(ctx, query, topicID, limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to list sub topics", "errors": fiber.Map{}})
		}
		defer rows.Close()

		type subTopicRow struct {
			ID        string `json:"id"`
			TopicID   string `json:"topicId"`
			SubjectID string `json:"subjectId"`
			Name      string `json:"name"`
			Code      string `json:"code"`
		}
		items := []subTopicRow{}
		for rows.Next() {
			var id, topicIDStr, name string
			var seq int
			if err := rows.Scan(&id, &topicIDStr, &name, &seq); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to scan sub topics", "errors": fiber.Map{}})
			}
			items = append(items, subTopicRow{
				ID:        id,
				TopicID:   topicIDStr,
				SubjectID: "",
				Name:      name,
				Code:      "",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"code":   "sub_topics_fetched",
			"items":  items,
			"total":  total,
			"errors": fiber.Map{},
		})
	}
}
