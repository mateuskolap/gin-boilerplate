package port

import "context"

type HealthChecker interface {
	// Readiness reports whether the application dependencies are ready to serve requests.
	Readiness(ctx context.Context) error
}
