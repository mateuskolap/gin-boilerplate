package port

import "context"

type CompromisedPasswordChecker interface {
	IsCompromised(ctx context.Context, password string) (bool, error)
}
