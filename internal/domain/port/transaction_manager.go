package port

import "context"

type TransactionManager interface {
	// Do runs fn within a transaction and commits it when fn succeeds.
	// An error from fn causes the transaction to be rolled back.
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
