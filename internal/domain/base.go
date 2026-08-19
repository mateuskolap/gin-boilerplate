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

type PaginationParams struct {
	Page     int
	Limit    int
	Preloads []string
}

type PaginatedResult[T any] struct {
	Items      []*T  `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

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
	OperatorIn                 FilterOperator = "IN"
	OperatorNotIn              FilterOperator = "NOT IN"
)

type Filter struct {
	Field    string
	Operator FilterOperator
	Value    interface{}
}

type BaseRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	GetByID(ctx context.Context, id uuid.UUID, preloads ...string) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params PaginationParams, filters []Filter) (*PaginatedResult[T], error)
}
