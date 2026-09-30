package shared

import (
	"context"
	"time"

	"uuid"
)

type BaseModel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BaseSoftDeleteModel struct {
	BaseModel
	DeletedAt *time.Time `json:"-"`
}

type BaseFindUseCase[T any] interface {
	// Find retrieves an entity by its UUID.
	Find(ctx context.Context, id uuid.UUID) (*T, error)
}

type BaseListUseCase[T any] interface {
	// List retrieves a paginated list of entities based on provided parameters and filters.
	List(ctx context.Context, params PaginationParams, filters []Filter) (*PaginatedResult[T], error)
}

type BaseDeleteUseCase interface {
	// Delete removes an entity by its UUID.
	Delete(ctx context.Context, id uuid.UUID) error
}
