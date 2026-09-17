package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

func topicsDetailHandler(pool *pgxpool.Pool) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := c.Context()
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "id required", "errors": fiber.Map{}})
		}

		// Topic details
		var topicID, subjectID, name string
		var seq int
		err := pool.QueryRow(ctx, `SELECT id, subject_id, name, sequence_index FROM topics WHERE id = $1`, id).Scan(&topicID, &subjectID, &name, &seq)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "topic not found", "errors": fiber.Map{}})
		}

		// Sub topics
		rows, err := pool.Query(ctx, `SELECT id, topic_id, name, sequence_index FROM sub_topics WHERE topic_id = $1 ORDER BY sequence_index ASC`, id)
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
		subTopics := []subTopicRow{}
		for rows.Next() {
			var sid, tid, sname string
			var sseq int
			if err := rows.Scan(&sid, &tid, &sname, &sseq); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to scan sub topics", "errors": fiber.Map{}})
			}
			subTopics = append(subTopics, subTopicRow{ID: sid, TopicID: tid, SubjectID: subjectID, Name: sname, Code: ""})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"id":          topicID,
			"subjectId":   subjectID,
			"name":        name,
			"code":        "",
			"description": "",
			"subTopics":   subTopics,
		})
	}
}
