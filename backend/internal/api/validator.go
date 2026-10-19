package api

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

// PaginationConfig holds pagination parameters with validation.
type PaginationConfig struct {
	Page  int
	Limit int
}

// ValidatePagination validates and normalizes pagination query parameters.
// Returns validated (page, limit) or error if invalid.
// Default: page=1, limit=50, maxLimit=100.
func ValidatePagination(c fiber.Ctx, maxLimit int) (PaginationConfig, error) {
	if maxLimit <= 0 {
		maxLimit = 100
	}

	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "50")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return PaginationConfig{}, ErrBadRequest("invalid page parameter", map[string][]string{"page": {"must be a positive integer"}})
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		return PaginationConfig{}, ErrBadRequest("invalid limit parameter", map[string][]string{"limit": {"must be a positive integer"}})
	}
	if limit > maxLimit {
		return PaginationConfig{}, ErrBadRequest("limit exceeds maximum", map[string][]string{"limit": {strconv.Itoa(maxLimit) + " is the maximum allowed"}})
	}

	return PaginationConfig{Page: page, Limit: limit}, nil
}

// ValidatePaginationWithDefaults validates pagination with custom defaults.
// If page/limit not provided, uses defaults.
func ValidatePaginationWithDefaults(c fiber.Ctx, defaultPage, defaultLimit, maxLimit int) (PaginationConfig, error) {
	if maxLimit <= 0 {
		maxLimit = 100
	}

	pageStr := c.Query("page")
	limitStr := c.Query("limit")

	page := defaultPage
	if pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 1 {
			return PaginationConfig{}, ErrBadRequest("invalid page parameter", map[string][]string{"page": {"must be a positive integer"}})
		}
		page = p
	}

	limit := defaultLimit
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil || l < 1 {
			return PaginationConfig{}, ErrBadRequest("invalid limit parameter", map[string][]string{"limit": {"must be a positive integer"}})
		}
		if l > maxLimit {
			return PaginationConfig{}, ErrBadRequest("limit exceeds maximum", map[string][]string{"limit": {strconv.Itoa(maxLimit) + " is the maximum allowed"}})
		}
		limit = l
	}

	return PaginationConfig{Page: page, Limit: limit}, nil
}

// GetLocalString safely extracts a string from c.Locals.
// Returns empty string and error if missing or wrong type.
func GetLocalString(c fiber.Ctx, key string) (string, error) {
	val, ok := c.Locals(key).(string)
	if !ok || val == "" {
		return "", errors.New("missing or invalid " + key)
	}
	return val, nil
}

// GetLocalUUID safely extracts and parses a UUID from c.Locals.
func GetLocalUUID(c fiber.Ctx, key string) (string, error) {
	val, err := GetLocalString(c, key)
	if err != nil {
		return "", err
	}
	// Validate UUID format
	if len(val) != 36 {
		return "", errors.New("invalid " + key + " format")
	}
	return val, nil
}

// ParseUUID parses a UUID string from request (path param, query, body).
func ParseUUID(s, fieldName string) (string, error) {
	if s == "" {
		return "", ErrBadRequest(fieldName+" is required", map[string][]string{fieldName: {"required"}})
	}
	if len(s) != 36 {
		return "", ErrBadRequest(fieldName+" must be a valid UUID", map[string][]string{fieldName: {"invalid uuid format"}})
	}
	return s, nil
}

// ValidateIDempotencyKey validates the Idempotency-Key header format.
// Requires RFC4122 UUID format.
func ValidateIDempotencyKey(c fiber.Ctx) (string, error) {
	key := c.Get("Idempotency-Key")
	if key == "" {
		return "", ErrBadRequest("Idempotency-Key header is required", map[string][]string{"Idempotency-Key": {"required"}})
	}
	// UUID v4 regex: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	if len(key) != 36 {
		return "", ErrBadRequest("Idempotency-Key must be a valid UUID", map[string][]string{"Idempotency-Key": {"must be a valid UUID"}})
	}
	// Basic UUID format check
	if key[8] != '-' || key[13] != '-' || key[18] != '-' || key[23] != '-' {
		return "", ErrBadRequest("Idempotency-Key must be a valid UUID", map[string][]string{"Idempotency-Key": {"invalid uuid format"}})
	}
	return key, nil
}

// PaginationMeta builds standard pagination metadata for responses.
func PaginationMeta(page, limit, total int) fiber.Map {
	return fiber.Map{
		"page":  page,
		"limit": limit,
		"total": total,
	}
}
