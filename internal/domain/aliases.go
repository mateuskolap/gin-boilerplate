package domain

import (
	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/domain/shared"
)

// Re-export port types
type TransactionManager = port.TransactionManager
type CacheRepository = port.CacheRepository

// Re-export shared base types
type BaseModel = shared.BaseModel
type BaseSoftDeleteModel = shared.BaseSoftDeleteModel
type BaseRepository[T any] = shared.BaseRepository[T]
type BaseFindUseCase[T any] = shared.BaseFindUseCase[T]
type BaseListUseCase[T any] = shared.BaseListUseCase[T]
type BaseDeleteUseCase = shared.BaseDeleteUseCase

// Re-export shared error types
type ErrorType = shared.ErrorType

const (
	ErrTypeNotFound        = shared.ErrTypeNotFound
	ErrTypeConflict        = shared.ErrTypeConflict
	ErrTypeUnauthorized    = shared.ErrTypeUnauthorized
	ErrTypeInternal        = shared.ErrTypeInternal
	ErrTypeValidation      = shared.ErrTypeValidation
	ErrTypeForbidden       = shared.ErrTypeForbidden
	ErrTypeTooManyRequests = shared.ErrTypeTooManyRequests
)

type AppError = shared.AppError
var NewAppError = shared.NewAppError

// Re-export shared pagination types
type PaginationParams = shared.PaginationParams
type PaginatedResult[T any] = shared.PaginatedResult[T]
type SortDirection = shared.SortDirection
type SortParam = shared.SortParam

const (
	SortAsc  = shared.SortAsc
	SortDesc = shared.SortDesc
)

// Re-export shared filter types
type Filter = shared.Filter
type FilterOperator = shared.FilterOperator
type Filters = shared.Filters

const (
	OperatorEquals             = shared.OperatorEquals
	OperatorNotEquals          = shared.OperatorNotEquals
	OperatorGreaterThan        = shared.OperatorGreaterThan
	OperatorLessThan           = shared.OperatorLessThan
	OperatorGreaterThanOrEqual = shared.OperatorGreaterThanOrEqual
	OperatorLessThanOrEqual    = shared.OperatorLessThanOrEqual
	OperatorLike               = shared.OperatorLike
	OperatorNotLike            = shared.OperatorNotLike
	OperatorILike              = shared.OperatorILike
	OperatorNotILike           = shared.OperatorNotILike
	OperatorIn                 = shared.OperatorIn
	OperatorNotIn              = shared.OperatorNotIn
	OperatorIsNull             = shared.OperatorIsNull
	OperatorIsNotNull          = shared.OperatorIsNotNull
)
