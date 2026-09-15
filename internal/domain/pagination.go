package domain

import "fmt"

type SortDirection string

const (
	SortAsc  SortDirection = "ASC"
	SortDesc SortDirection = "DESC"
)

type SortParam struct {
	Field     string
	Direction SortDirection
}

type PaginationParams struct {
	Page  int
	Limit int
	Sort  []SortParam
}

type PaginatedResult[T any] struct {
	Items      []*T  `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// Sanitize normalizes pagination parameters with safe fallback defaults.
func (p *PaginationParams) Sanitize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 10
	}
}

// ValidateSort checks if all sort fields belong to the allowedFields set and have valid directions.
func (p *PaginationParams) ValidateSort(allowedFields map[string]bool) error {
	for _, s := range p.Sort {
		if !allowedFields[s.Field] {
			return NewAppError(
				ErrTypeValidation,
				fmt.Sprintf("sorting by field '%s' is not allowed", s.Field),
				nil,
			)
		}
		if s.Direction != SortAsc && s.Direction != SortDesc {
			return NewAppError(
				ErrTypeValidation,
				fmt.Sprintf("invalid sort direction: '%s'", s.Direction),
				nil,
			)
		}
	}
	return nil
}

// Offset calculates the database record offset based on sanitized Page and Limit.
func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.Limit
}
