package api

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

func subjectsListHandler(pool *pgxpool.Pool) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := c.Context()
		page, _ := strconv.Atoi(c.Query("page", "1"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(c.Query("limit", "50"))
		if limit < 1 || limit > 200 {
			limit = 50
		}
		offset := (page - 1) * limit
		search := c.Query("search")
		grade := c.Query("grade")

		baseQuery := `SELECT s.id, s.grade_level_id, s.name, s.code, COALESCE(gl.local_label, '') FROM subjects s LEFT JOIN grade_levels gl ON gl.id = s.grade_level_id`
		countQuery := `SELECT COUNT(*) FROM subjects s LEFT JOIN grade_levels gl ON gl.id = s.grade_level_id`
		conds := []string{}
		args := []interface{}{}
		argIdx := 1

		if strings.TrimSpace(search) != "" {
			conds = append(conds, `(s.name ILIKE $`+strconv.Itoa(argIdx)+` OR s.code ILIKE $`+strconv.Itoa(argIdx)+`)`)
			args = append(args, "%"+search+"%")
			argIdx++
		}
		if strings.TrimSpace(grade) != "" && grade != "all" {
			conds = append(conds, `gl.local_label = $`+strconv.Itoa(argIdx))
			args = append(args, grade)
			argIdx++
		}

		where := ""
		if len(conds) > 0 {
			where = " WHERE " + strings.Join(conds, " AND ")
		}

		countSQL := countQuery + where
		var total int
		if err := pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to count subjects", "errors": fiber.Map{}})
		}

		query := baseQuery + where + ` ORDER BY s.name ASC LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
		args = append(args, limit, offset)

		rows, err := pool.Query(ctx, query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to list subjects", "errors": fiber.Map{}})
		}
		defer rows.Close()

		type subjectRow struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Code    string `json:"code"`
			Grade   string `json:"grade"`
			GradeID string `json:"gradeId"`
		}
		items := []subjectRow{}
		for rows.Next() {
			var r subjectRow
			if err := rows.Scan(&r.ID, &r.GradeID, &r.Name, &r.Code, &r.Grade); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to scan subjects", "errors": fiber.Map{}})
			}
			items = append(items, r)
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"code":   "subjects_fetched",
			"items":  items,
			"total":  total,
			"errors": fiber.Map{},
		})
	}
}
