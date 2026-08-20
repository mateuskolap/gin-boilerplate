package dto

import "gin-boilerplate/internal/domain"

type PaginatedResponse[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// ToPaginatedResponse converts a domain PaginatedResult[D] into a DTO PaginatedResponse[R]
// using the provided mapper function.
func ToPaginatedResponse[D any, R any](
	result *domain.PaginatedResult[D],
	mapper func(*D) R,
) PaginatedResponse[R] {
	if result == nil {
		return PaginatedResponse[R]{}
	}

	items := make([]R, len(result.Items))
	for i, item := range result.Items {
		items[i] = mapper(item)
	}

	return PaginatedResponse[R]{
		Items:      items,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}
}
