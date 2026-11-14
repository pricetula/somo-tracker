package cursor

import (
	"net/url"
	"testing"

	"somotracker/internal/platform/apperr"
)

func defaultConfig() *ListConfig {
	return &ListConfig{
		DefaultSort: "name_asc",
		DefaultLimit: 50,
		MaxLimit: 100,
		SortFields: []SortField{
			{
				Name: "name_asc",
				Columns: []SortColumn{
					{Column: "full_name", Asc: true, NullsLast: true},
					{Column: "student_id", Asc: true, NullsLast: true},
				},
			},
			{
				Name: "name_desc",
				Columns: []SortColumn{
					{Column: "full_name", Asc: false, NullsLast: true},
					{Column: "student_id", Asc: true, NullsLast: true},
				},
			},
		},
		FilterFields: []FilterField{
			{
				Name:     "fullName",
				Column:   "full_name",
				Operators: []Operator{OpEq, OpContains, OpIsNull},
			},
			{
				Name:     "gender",
				Column:   "gender",
				Operators: []Operator{OpEq, OpIn, OpIsNull},
			},
		},
	}
}

func TestParseListParams_Defaults(t *testing.T) {
	cfg := defaultConfig()
	query := url.Values{}
	params, err := ParseListParams(query, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Limit != 50 {
		t.Errorf("expected limit 50, got %d", params.Limit)
	}
	if params.Sort != "name_asc" {
		t.Errorf("expected sort name_asc, got %s", params.Sort)
	}
	if params.Cursor != nil {
		t.Errorf("expected nil cursor, got %v", params.Cursor)
	}
	if len(params.Filters) != 0 {
		t.Errorf("expected no filters, got %v", params.Filters)
	}
}

func TestParseListParams_InvalidLimit(t *testing.T) {
	cfg := defaultConfig()

	t.Run("non-numeric", func(t *testing.T) {
		query := url.Values{"limit": {"abc"}}
		_, err := ParseListParams(query, cfg)
		if err == nil || err.(*apperr.Error).Code != apperr.BAD_REQUEST {
			t.Errorf("expected BAD_REQUEST, got %v", err)
		}
	})

	t.Run("below minimum", func(t *testing.T) {
		query := url.Values{"limit": {"0"}}
		_, err := ParseListParams(query, cfg)
		if err == nil || err.(*apperr.Error).Code != apperr.BAD_REQUEST {
			t.Errorf("expected BAD_REQUEST, got %v", err)
		}
	})

	t.Run("above max", func(t *testing.T) {
		query := url.Values{"limit": {"200"}}
		_, err := ParseListParams(query, cfg)
		if err == nil || err.(*apperr.Error).Code != apperr.BAD_REQUEST {
			t.Errorf("expected BAD_REQUEST, got %v", err)
		}
	})
}

func TestParseListParams_InvalidSort(t *testing.T) {
	cfg := defaultConfig()
	query := url.Values{"sort": {"invalid_sort"}}
	_, err := ParseListParams(query, cfg)
	if err == nil || err.(*apperr.Error).Code != apperr.BAD_REQUEST {
		t.Errorf("expected BAD_REQUEST, got %v", err)
	}
}

func TestParseListParams_UnknownFilter(t *testing.T) {
	cfg := defaultConfig()
	query := url.Values{"unknownField": {"eq:value"}}
	_, err := ParseListParams(query, cfg)
	// Unknown filter fields are simply ignored (not in allowlist)
	if err != nil {
		t.Errorf("unexpected error for unknown filter: %v", err)
	}
}

func TestParseListParams_DisallowedOperator(t *testing.T) {
	cfg := defaultConfig()
	// gender doesn't allow "contains"
	query := url.Values{"gender": {"contains:male"}}
	_, err := ParseListParams(query, cfg)
	if err == nil || err.(*apperr.Error).Code != apperr.BAD_REQUEST {
		t.Errorf("expected BAD_REQUEST for disallowed operator, got %v", err)
	}
}

func TestParseListParams_ValidFilters(t *testing.T) {
	cfg := defaultConfig()
	query := url.Values{
		"fullName": {"contains:john"},
		"gender":   {"eq:male"},
	}
	params, err := ParseListParams(query, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params.Filters) != 2 {
		t.Errorf("expected 2 filters, got %d", len(params.Filters))
	}
	fv, ok := params.Filters["fullName"]
	if !ok {
		t.Fatal("expected fullName filter")
	}
	if fv.Op != OpContains || fv.Val != "john" {
		t.Errorf("expected OpContains john, got %v:%v", fv.Op, fv.Val)
	}
}

func TestParseListParams_CursorDecodeError(t *testing.T) {
	cfg := defaultConfig()
	query := url.Values{"cursor": {"not-valid-base64!!!"}}
	_, err := ParseListParams(query, cfg)
	if err == nil || err.(*apperr.Error).Code != apperr.INVALID_CURSOR {
		t.Errorf("expected INVALID_CURSOR, got %v", err)
	}
}

func TestParseListParams_CursorContextMismatch(t *testing.T) {
	cfg := defaultConfig()
	// Create a cursor for name_asc, then try to use it with name_desc
	cursor, err := EncodeCursor(&Cursor{
		Sort:       []string{"name_asc"},
		Last:       []interface{}{"Alice", "uuid-123"},
		Tiebreaker: "uuid-123",
		Ctx:        "somehash",
	})
	if err != nil {
		t.Fatalf("failed to encode cursor: %v", err)
	}
	query := url.Values{
		"sort":   {"name_desc"},
		"cursor": {cursor},
	}
	_, err = ParseListParams(query, cfg)
	if err == nil || err.(*apperr.Error).Code != apperr.INVALID_CURSOR {
		t.Errorf("expected INVALID_CURSOR for context mismatch, got %v", err)
	}
}

func TestCursorEncodeDecodeRoundTrip(t *testing.T) {
	original := &Cursor{
		Sort:       []string{"name_asc"},
		Last:       []interface{}{"Alice", "uuid-123"},
		Tiebreaker: "uuid-123",
		Ctx:        "abc123",
	}
	encoded, err := EncodeCursor(original)
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if decoded.Sort[0] != original.Sort[0] {
		t.Errorf("sort mismatch: %v vs %v", decoded.Sort, original.Sort)
	}
	if decoded.Tiebreaker != original.Tiebreaker {
		t.Errorf("tiebreaker mismatch: %v vs %v", decoded.Tiebreaker, original.Tiebreaker)
	}
	if decoded.Ctx != original.Ctx {
		t.Errorf("ctx mismatch: %v vs %v", decoded.Ctx, original.Ctx)
	}
}

func TestCursorContextHashMatches(t *testing.T) {
	filters := map[string]FilterValue{
		"fullName": {Op: OpContains, Val: "john"},
		"gender":   {Op: OpEq, Val: "male"},
	}
	hash := ContextHash("name_asc", filters)

	c := &Cursor{Ctx: hash}
	if !c.ContextHashMatches("name_asc", filters) {
		t.Error("expected context hash to match")
	}

	// Different sort should not match
	if c.ContextHashMatches("name_desc", filters) {
		t.Error("expected context hash to NOT match for different sort")
	}

	// Different filters should not match
	diffFilters := map[string]FilterValue{
		"fullName": {Op: OpContains, Val: "jane"},
		"gender":   {Op: OpEq, Val: "male"},
	}
	if c.ContextHashMatches("name_asc", diffFilters) {
		t.Error("expected context hash to NOT match for different filters")
	}
}

func TestFilterField_OpAllowed(t *testing.T) {
	ff := &FilterField{
		Name:     "gender",
		Column:   "gender",
		Operators: []Operator{OpEq, OpIn, OpIsNull},
	}
	if !ff.OpAllowed(OpEq) {
		t.Error("expected OpEq to be allowed")
	}
	if !ff.OpAllowed(OpIn) {
		t.Error("expected OpIn to be allowed")
	}
	if !ff.OpAllowed(OpIsNull) {
		t.Error("expected OpIsNull to be allowed")
	}
	if ff.OpAllowed(OpContains) {
		t.Error("expected OpContains to NOT be allowed")
	}
	if ff.OpAllowed(OpBetween) {
		t.Error("expected OpBetween to NOT be allowed")
	}
}

func TestParseFilterValue(t *testing.T) {
	t.Run("eq", func(t *testing.T) {
		val, err := parseFilterValue(OpEq, "hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "hello" {
			t.Errorf("expected 'hello', got %v", val)
		}
	})

	t.Run("isNull true", func(t *testing.T) {
		val, err := parseFilterValue(OpIsNull, "true")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != true {
			t.Errorf("expected true, got %v", val)
		}
	})

	t.Run("in", func(t *testing.T) {
		val, err := parseFilterValue(OpIn, "a,b,c")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		arr, ok := val.([]string)
		if !ok || len(arr) != 3 {
			t.Errorf("expected []string{len 3}, got %v", val)
		}
	})

	t.Run("between", func(t *testing.T) {
		val, err := parseFilterValue(OpBetween, "2000-01-01,2000-12-31")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		parts, ok := val.([]string)
		if !ok || len(parts) != 2 {
			t.Errorf("expected []string{len 2}, got %v", val)
		}
	})

	t.Run("between invalid", func(t *testing.T) {
		_, err := parseFilterValue(OpBetween, "only-one")
		if err == nil {
			t.Error("expected error for between with one value")
		}
	})
}

func TestListConfig_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cfg := defaultConfig()
		if err := cfg.Validate(); err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})

	t.Run("defaultLimit above max", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.DefaultLimit = 200
		cfg.MaxLimit = 100
		if err := cfg.Validate(); err == nil {
			t.Error("expected validation error for defaultLimit > maxLimit")
		}
	})

	t.Run("too many sort columns", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.SortFields = []SortField{
			{
				Name: "bad_sort",
				Columns: []SortColumn{
					{Column: "a", Asc: true},
					{Column: "b", Asc: true},
					{Column: "c", Asc: true},
					{Column: "d", Asc: true},
				},
			},
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected validation error for >3 sort columns")
		}
	})

	t.Run("duplicate sort name", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.SortFields = append(cfg.SortFields, cfg.SortFields[0])
		if err := cfg.Validate(); err == nil {
			t.Error("expected validation error for duplicate sort name")
		}
	})
}
