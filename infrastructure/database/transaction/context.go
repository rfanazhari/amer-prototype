// Package transaction shares an infrastructure-owned transaction between persistence adapters.
package transaction

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

// key keeps the transaction carrier private to infrastructure.
type key struct{}

// Context binds a caller-owned transaction to a callback context.
func Context(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, key{}, tx)
}

// Session returns the joined transaction or the pool, always bound to the current context.
func Session(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(key{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}

// Within joins an existing transaction or opens one with optional isolation settings.
func Within(ctx context.Context, db *gorm.DB, fn func(context.Context, *gorm.DB) error, opts ...*sql.TxOptions) error {
	if tx, ok := ctx.Value(key{}).(*gorm.DB); ok {
		return fn(ctx, tx.WithContext(ctx))
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(Context(ctx, tx), tx) }, opts...)
}
