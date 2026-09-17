package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

func subjectsDetailHandler(pool *pgxpool.Pool) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := c.Context()
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "id required", "errors": fiber.Map{}})
		}

		// Subject details
		var subjID, subjName, subjCode, grade string
		err := pool.QueryRow(ctx, `SELECT s.id, s.name, s.code, COALESCE(gl.local_label,'') FROM subjects s LEFT JOIN grade_levels gl ON gl.id = s.grade_level_id WHERE s.id = $1`, id).Scan(&subjID, &subjName, &subjCode, &grade)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "subject not found", "errors": fiber.Map{}})
		}

		// Topics for subject
		rows, err := pool.Query(ctx, `SELECT id, subject_id, name, sequence_index FROM topics WHERE subject_id = $1 ORDER BY sequence_index ASC`, id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to list topics", "errors": fiber.Map{}})
		}
		defer rows.Close()
		type topicRow struct {
			ID          string `json:"id"`
			SubjectID   string `json:"subjectId"`
			Name        string `json:"name"`
			Code        string `json:"code"`
			Description string `json:"description"`
		}
		topics := []topicRow{}
		for rows.Next() {
			var tid, sid, name string
			var seq int
			if err := rows.Scan(&tid, &sid, &name, &seq); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to scan topics", "errors": fiber.Map{}})
			}
			topics = append(topics, topicRow{ID: tid, SubjectID: sid, Name: name, Code: "", Description: ""})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"id":     subjID,
			"name":   subjName,
			"code":   subjCode,
			"grade":  grade,
			"topics": topics,
		})
	}
}
