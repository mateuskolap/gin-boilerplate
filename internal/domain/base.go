package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BaseSoftDeleteModel struct {
	BaseModel
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type BaseRepository[T any] interface {
	// Create persists a new entity in the database.
	Create(ctx context.Context, entity *T) error

	// GetByID fetches an entity by its UUID, optionally preloading relationships.
	// Returns (nil, nil) if the record does not exist.
	GetByID(ctx context.Context, id uuid.UUID, preloads ...string) (*T, error)

	// Update saves changes made to an existing entity.
	Update(ctx context.Context, entity *T) error

	// Delete removes an entity by its UUID (or soft-deletes if supported).
	Delete(ctx context.Context, id uuid.UUID) error

	// List queries a paginated slice of entities applying preloads, filters, and sorting.
	List(ctx context.Context, params PaginationParams, filters []Filter) (*PaginatedResult[T], error)
}
