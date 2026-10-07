package application_test

import (
	"context"
	"testing"

	"gin-boilerplate/internal/domain/shared"
	organizationapp "gin-boilerplate/internal/organizations/application"
	organizationdomain "gin-boilerplate/internal/organizations/domain"
)

type organizationListRepo struct {
	organizationdomain.OrganizationRepository
	called bool
}

func (r *organizationListRepo) List(_ context.Context, _ shared.PaginationParams, _ []shared.Filter) (*shared.PaginatedResult[organizationdomain.Organization], error) {
	r.called = true
	return &shared.PaginatedResult[organizationdomain.Organization]{Page: 1, Limit: 10}, nil
}

func TestOrganizationListAllowsNameFilter(t *testing.T) {
	repo := &organizationListRepo{}
	filters := []shared.Filter{{Field: "name", Operator: shared.OperatorLike, Value: "%acme%"}}
	result, err := organizationapp.NewOrganizationUseCase(repo).List(context.Background(), shared.PaginationParams{Page: 1, Limit: 10}, filters)
	if err != nil || result == nil || !repo.called {
		t.Fatalf("List() result=%+v called=%v error=%v", result, repo.called, err)
	}
}
