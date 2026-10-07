package api

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"somotracker/backend/internal/services"
)

func subjectsListHandler(curriculumSvc services.CurriculumService) fiber.Handler {
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
		search := c.Query("search")
		grade := c.Query("grade")

		items, total, err := curriculumSvc.ListSubjects(ctx, page, limit, search, grade)
		if err != nil {
			if strings.Contains(err.Error(), "bad_request:") {
				return WriteError(c, ErrBadRequest(err.Error(), nil))
			}
			return WriteError(c, ErrInternal(err.Error()))
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"code":   "subjects_fetched",
			"items":  items,
			"total":  total,
			"errors": fiber.Map{},
		})
	}
}
