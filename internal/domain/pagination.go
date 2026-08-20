package domain

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
	Page     int
	Limit    int
	Sort     []SortParam
	Preloads []string
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

// Offset calculates the database record offset based on sanitized Page and Limit.
func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.Limit
}
