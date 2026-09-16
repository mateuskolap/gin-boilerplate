package repository

import (
	"context"
	"gin-boilerplate/internal/domain"

	"gorm.io/gorm"
)

type txKey struct{}

type gormTransactionManagerRepository struct {
	db *gorm.DB
}

func NewGormTransactionManagerRepository(db *gorm.DB) domain.TransactionManager {
	return &gormTransactionManagerRepository{
		db: db,
	}
}

func (tm *gormTransactionManagerRepository) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctxWithTx := context.WithValue(ctx, txKey{}, tx)
		return fn(ctxWithTx)
	})
}

func GetTxFromContext(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	if ok {
		return tx
	}
	return defaultDB.WithContext(ctx)
}
