package api

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

func topicsListHandler(pool *pgxpool.Pool) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := c.Context()
		subjectID := c.Query("subject_id")
		if strings.TrimSpace(subjectID) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "subject_id is required", "errors": fiber.Map{}})
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

		countSQL := `SELECT COUNT(*) FROM topics WHERE subject_id = $1`
		var total int
		if err := pool.QueryRow(ctx, countSQL, subjectID).Scan(&total); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to count topics", "errors": fiber.Map{}})
		}

		query := `SELECT id, subject_id, name, sequence_index FROM topics WHERE subject_id = $1 ORDER BY sequence_index ASC LIMIT $2 OFFSET $3`
		rows, err := pool.Query(ctx, query, subjectID, limit, offset)
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
		items := []topicRow{}
		for rows.Next() {
			var id, subjectIDStr, name string
			var seq int
			if err := rows.Scan(&id, &subjectIDStr, &name, &seq); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to scan topics", "errors": fiber.Map{}})
			}
			items = append(items, topicRow{
				ID:          id,
				SubjectID:   subjectIDStr,
				Name:        name,
				Code:        "",
				Description: "",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"code":   "topics_fetched",
			"items":  items,
			"total":  total,
			"errors": fiber.Map{},
		})
	}
}
