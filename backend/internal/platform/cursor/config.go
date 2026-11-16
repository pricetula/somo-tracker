package cursor

import "fmt"

// ListConfig declaratively configures keyset pagination for a list endpoint.
// One ListConfig per endpoint, declared by the owning domain.
type ListConfig struct {
	// DefaultSort is the default sort name (e.g. "name_asc").
	DefaultSort string

	// DefaultLimit is the default page size (BE-014: 50).
	DefaultLimit int

	// MaxLimit is the maximum allowed page size (BE-014: 100).
	MaxLimit int

	// SortFields declares whitelisted sorts. Each sort name maps to a sqlc query
	// with a hardcoded keyset predicate.
	SortFields []SortField

	// FilterFields declares whitelisted filter fields and allowed operators.
	FilterFields []FilterField
}

// SortField declares a whitelisted sort with its composite sort key columns.
type SortField struct {
	// Name is the sort value accepted in the `sort` query param (e.g. "name_asc").
	Name string

	// Columns is the composite sort key (1-3 columns).
	Columns []SortColumn
}

// SortColumn is a single column in a composite sort key.
type SortColumn struct {
	Column   string // DB column name
	Asc      bool   // sort direction: true = ASC, false = DESC
	NullsLast bool   // NULLS LAST handling
}

// FilterField declares a whitelisted filter field and its allowed operators.
type FilterField struct {
	// Name is the query param name (camelCase, e.g. "fullName").
	Name string

	// Column is the DB column name (snake_case, e.g. "full_name").
	Column string

	// Operators is the allowlist of operators for this field.
	Operators []Operator
}

// Operator is a filter operator.
type Operator string

const (
	OpEq      Operator = "eq"      // =
	OpIn      Operator = "in"      // IN (...)
	OpBetween Operator = "between" // BETWEEN ... AND ...
	OpContains Operator = "contains" // ILIKE %...%
	OpIsNull  Operator = "is_null" // IS NULL / IS NOT NULL
)

// Validate checks the ListConfig for consistency and panics on invalid config.
// Called during package init or service startup.
func (c *ListConfig) Validate() error {
	if c.DefaultLimit < 1 {
		return fmt.Errorf("default limit must be >= 1, got %d", c.DefaultLimit)
	}
	if c.MaxLimit < 1 {
		return fmt.Errorf("max limit must be >= 1, got %d", c.MaxLimit)
	}
	if c.DefaultLimit > c.MaxLimit {
		return fmt.Errorf("default limit %d exceeds max limit %d", c.DefaultLimit, c.MaxLimit)
	}

	sortNames := make(map[string]bool)
	for _, sf := range c.SortFields {
		if sortNames[sf.Name] {
			return fmt.Errorf("duplicate sort name: %s", sf.Name)
		}
		sortNames[sf.Name] = true
		if len(sf.Columns) < 1 || len(sf.Columns) > 3 {
			return fmt.Errorf("sort %s must have 1-3 columns, got %d", sf.Name, len(sf.Columns))
		}
	}

	filterNames := make(map[string]bool)
	for _, ff := range c.FilterFields {
		if filterNames[ff.Name] {
			return fmt.Errorf("duplicate filter field: %s", ff.Name)
		}
		filterNames[ff.Name] = true
		if len(ff.Operators) == 0 {
			return fmt.Errorf("filter %s must have at least one operator", ff.Name)
		}
	}

	return nil
}

// SortByName returns the SortField for the given sort name, or nil if not found.
func (c *ListConfig) SortByName(name string) *SortField {
	for i := range c.SortFields {
		if c.SortFields[i].Name == name {
			return &c.SortFields[i]
		}
	}
	return nil
}

// FilterByName returns the FilterField for the given filter name, or nil if not found.
func (c *ListConfig) FilterByName(name string) *FilterField {
	for i := range c.FilterFields {
		if c.FilterFields[i].Name == name {
			return &c.FilterFields[i]
		}
	}
	return nil
}
