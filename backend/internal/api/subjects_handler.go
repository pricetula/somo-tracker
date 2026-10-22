package api

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"somotracker/backend/internal/services"
)

func subjectsListHandler(curriculumSvc services.CurriculumService) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := c.Context()

		pagination, err := ValidatePagination(c, 100)
		if err != nil {
			return err
		}
		page, limit := pagination.Page, pagination.Limit

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
			"page":   page,
			"limit":  limit,
			"errors": fiber.Map{},
		})
	}
}
