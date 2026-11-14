package cursor

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"somotracker/internal/platform/apperr"
)

// ListParams is the parsed and validated result of query parameters.
type ListParams struct {
	Limit   int
	Sort    string
	Cursor  *Cursor
	Filters map[string]FilterValue
}

// FilterValue is a parsed filter value with its operator.
type FilterValue struct {
	Op  Operator
	Val interface{}
}

// ParseListParams parses and validates query parameters against a ListConfig.
// Returns BAD_REQUEST or INVALID_CURSOR errors via apperr.
func ParseListParams(query url.Values, cfg *ListConfig) (*ListParams, error) {
	// Parse limit
	limit := cfg.DefaultLimit
	if limitStr := query.Get("limit"); limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil || l < 1 {
			return nil, apperr.New(apperr.BAD_REQUEST, "invalid limit")
		}
		if l > cfg.MaxLimit {
			return nil, apperr.New(apperr.BAD_REQUEST, "limit exceeds maximum")
		}
		limit = l
	}

	// Parse sort
	sort := cfg.DefaultSort
	if sortStr := query.Get("sort"); sortStr != "" {
		if cfg.SortByName(sortStr) == nil {
			return nil, apperr.New(apperr.BAD_REQUEST, "invalid sort")
		}
		sort = sortStr
	}

	// Parse filters
	filters := make(map[string]FilterValue)
	for _, ff := range cfg.FilterFields {
		paramName := ff.Name
		val := query.Get(paramName)
		if val == "" {
			continue
		}
		opStr, valStr, ok := strings.Cut(val, ":")
		if !ok {
			return nil, apperr.New(apperr.BAD_REQUEST, "invalid filter format for "+paramName)
		}
		op := Operator(opStr)
		if !ff.OpAllowed(op) {
			return nil, apperr.New(apperr.BAD_REQUEST, "invalid operator for "+paramName)
		}
		parsedVal, err := parseFilterValue(op, valStr)
		if err != nil {
			return nil, apperr.New(apperr.BAD_REQUEST, "invalid filter value for "+paramName)
		}
		filters[paramName] = FilterValue{Op: op, Val: parsedVal}
	}

	// Parse cursor
	var cursor *Cursor
	if cursorStr := query.Get("cursor"); cursorStr != "" {
		c, err := DecodeCursor(cursorStr)
		if err != nil {
			return nil, apperr.New(apperr.INVALID_CURSOR, "invalid cursor")
		}
		// Validate context hash matches current sort+filters
		if !c.ContextHashMatches(sort, filters) {
			return nil, apperr.New(apperr.INVALID_CURSOR, "cursor context mismatch")
		}
		cursor = c
	}

	return &ListParams{
		Limit:   limit,
		Sort:    sort,
		Cursor:  cursor,
		Filters: filters,
	}, nil
}

// OpAllowed checks if the operator is in the allowlist.
func (ff *FilterField) OpAllowed(op Operator) bool {
	for _, o := range ff.Operators {
		if o == op {
			return true
		}
	}
	return false
}

// parseFilterValue parses a filter value string based on the operator.
func parseFilterValue(op Operator, valStr string) (interface{}, error) {
	switch op {
	case OpIsNull:
		b, err := strconv.ParseBool(valStr)
		if err != nil {
			return nil, err
		}
		return b, nil
	case OpIn:
		return strings.Split(valStr, ","), nil
	case OpBetween:
		parts := strings.Split(valStr, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("between requires two values")
		}
		return parts, nil
	default: // eq, contains
		return valStr, nil
	}
}

// FilterWire is the wire format for a filter value (snake_case field names).
type FilterWire struct {
	Op  Operator     `json:"op"`
	Val interface{} `json:"v"`
}
