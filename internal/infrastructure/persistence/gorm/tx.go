package gormstore

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type txContextKey struct{}

type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

func (m *TxManager) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txContextKey{}, tx)
		if err := fn(txCtx); err != nil {
			return err
		}
		return nil
	})
}

func DBFromContext(ctx context.Context, root *gorm.DB) *gorm.DB {
	tx, ok := ctx.Value(txContextKey{}).(*gorm.DB)
	if ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return root.WithContext(ctx)
}

func MustDBFromContext(ctx context.Context) (*gorm.DB, error) {
	tx, ok := ctx.Value(txContextKey{}).(*gorm.DB)
	if !ok || tx == nil {
		return nil, fmt.Errorf("transaction db not found in context")
	}
	return tx.WithContext(ctx), nil
}
