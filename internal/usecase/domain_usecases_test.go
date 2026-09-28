package usecase

import (
	"context"
	"testing"
	"uuid"

	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
)

type testAuthorizationRepo struct {
	userID     uuid.UUID
	permission domain.PermissionName
	allowed    bool
	err        error
}

func (r *testAuthorizationRepo) UserHasPermission(_ context.Context, userID uuid.UUID, permission domain.PermissionName) (bool, error) {
	r.userID, r.permission = userID, permission
	return r.allowed, r.err
}

func TestPermissionCheckerDelegatesCurrentPermissionLookup(t *testing.T) {
	userID := uuid.New()
	repo := &testAuthorizationRepo{allowed: true}
	allowed, err := NewPermissionCheckerUseCase(repo).HasPermission(context.Background(), userID, domain.PermissionViewUser)
	if err != nil || !allowed || repo.userID != userID || repo.permission != domain.PermissionViewUser {
		t.Fatalf("HasPermission() allowed=%v userID=%v permission=%q error=%v", allowed, repo.userID, repo.permission, err)
	}

	wantErr := context.DeadlineExceeded
	repo.err = wantErr
	if _, err := NewPermissionCheckerUseCase(repo).HasPermission(context.Background(), userID, domain.PermissionViewUser); err != wantErr {
		t.Fatalf("HasPermission() error=%v, want original error %v", err, wantErr)
	}
}

type testPermissionRepo struct {
	domain.PermissionRepository
	result   *shared.PaginatedResult[domain.Permission]
	err      error
	params   shared.PaginationParams
	filters  []shared.Filter
	preloads []string
	calls    int
}

func (r *testPermissionRepo) List(_ context.Context, params shared.PaginationParams, filters []shared.Filter, preloads ...string) (*shared.PaginatedResult[domain.Permission], error) {
	r.calls++
	r.params, r.filters, r.preloads = params, filters, preloads
	return r.result, r.err
}

func TestPermissionUseCaseAllowsOnlyPublicFilters(t *testing.T) {
	result := &shared.PaginatedResult[domain.Permission]{Page: 1, Limit: 10}
	repo := &testPermissionRepo{result: result}
	useCase := NewPermissionUseCase(repo)
	params := shared.PaginationParams{Page: 1, Limit: 10, Sort: []shared.SortParam{{Field: "name", Direction: shared.SortAsc}}}
	filters := []shared.Filter{{Field: "name", Operator: shared.OperatorEquals, Value: "view_user"}}

	got, err := useCase.List(context.Background(), params, filters)
	if err != nil || got != result || repo.calls != 1 || repo.params.Sort[0].Field != "name" || len(repo.filters) != 1 {
		t.Fatalf("List() result=%p calls=%d params=%+v filters=%+v error=%v", got, repo.calls, repo.params, repo.filters, err)
	}

	if _, err := useCase.List(context.Background(), params, []shared.Filter{{Field: "password", Operator: shared.OperatorEquals}}); appErrorType(err) != shared.ErrTypeValidation || repo.calls != 1 {
		t.Fatalf("List() accepted a private filter or queried repository: error=%v calls=%d", err, repo.calls)
	}
}
