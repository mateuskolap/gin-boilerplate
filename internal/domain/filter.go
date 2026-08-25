package domain

import (
	"fmt"
	"slices"
)

type FilterOperator string

const (
	OperatorEquals             FilterOperator = "="
	OperatorNotEquals          FilterOperator = "!="
	OperatorGreaterThan        FilterOperator = ">"
	OperatorLessThan           FilterOperator = "<"
	OperatorGreaterThanOrEqual FilterOperator = ">="
	OperatorLessThanOrEqual    FilterOperator = "<="
	OperatorLike               FilterOperator = "LIKE"
	OperatorNotLike            FilterOperator = "NOT LIKE"
	OperatorILike              FilterOperator = "ILIKE"
	OperatorNotILike           FilterOperator = "NOT ILIKE"
	OperatorIn                 FilterOperator = "IN"
	OperatorNotIn              FilterOperator = "NOT IN"
)

var validOperators = map[FilterOperator]bool{
	OperatorEquals: true, OperatorNotEquals: true,
	OperatorGreaterThan: true, OperatorLessThan: true,
	OperatorGreaterThanOrEqual: true, OperatorLessThanOrEqual: true,
	OperatorLike: true, OperatorNotLike: true,
	OperatorILike: true, OperatorNotILike: true,
	OperatorIn: true, OperatorNotIn: true,
}

type Filter struct {
	Field    string
	Operator FilterOperator
	Value    interface{}
}

// Validate checks whether the filter uses a recognized and safe operator.
func (f Filter) Validate() error {
	if !validOperators[f.Operator] {
		return fmt.Errorf("invalid filter operator: %q", f.Operator)
	}
	return nil
}

// IsSetOperator returns true if the operator requires set syntax (IN, NOT IN).
func (f Filter) IsSetOperator() bool {
	return f.Operator == OperatorIn || f.Operator == OperatorNotIn
}

type Filters []Filter

// Without returns a new copy of Filters excluding any filter whose Field matches one of the given fields.
func (f Filters) Without(fields ...string) Filters {
	return slices.DeleteFunc(slices.Clone(f), func(filter Filter) bool {
		return slices.Contains(fields, filter.Field)
	})
}
