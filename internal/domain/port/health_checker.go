package port

import "context"

type HealthChecker interface {
	Readiness(ctx context.Context) error
}
