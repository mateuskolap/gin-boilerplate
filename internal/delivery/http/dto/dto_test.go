package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"uuid"

	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
)

func TestEntityResponseMappersExposeOnlyPublicFields(t *testing.T) {
	userID, roleID, permissionID, tokenID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	created := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	user := &domain.User{
		ID:        userID,
		CreatedAt: created,
		UpdatedAt: created,
		Name:      "Alice",
		Email:     "alice@example.com",
		Password:  "secret-hash",
		Roles:     []domain.Role{{ID: roleID, Name: "Admin"}},
	}
	userResponse := ToUserResponse(user)
	if userResponse.ID != userID || userResponse.Name != "Alice" || userResponse.Email != "alice@example.com" || userResponse.CreatedAt != created {
		t.Fatalf("ToUserResponse() = %+v", userResponse)
	}
	userWithRole := ToUserWithRoleResponse(user)
	if userWithRole.ID != userID || len(userWithRole.Roles) != 1 || userWithRole.Roles[0].ID != roleID {
		t.Fatalf("ToUserWithRoleResponse() = %+v", userWithRole)
	}
	encoded, err := json.Marshal(userWithRole)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "secret-hash") {
		t.Fatalf("user response exposed password: %s", encoded)
	}

	role := &domain.Role{ID: roleID, Name: "Editor", Permissions: []domain.Permission{{ID: permissionID, Name: "view_user"}}}
	if got := ToRoleResponse(role); got.ID != roleID || got.Name != "Editor" {
		t.Fatalf("ToRoleResponse() = %+v", got)
	}
	if got := ToRoleWithPermissionsResponse(role); len(got.Permissions) != 1 || got.Permissions[0].ID != permissionID || got.Permissions[0].Name != "view_user" {
		t.Fatalf("ToRoleWithPermissionsResponse() = %+v", got)
	}
	if got := ToPermissionResponse(&domain.Permission{ID: permissionID, Name: "view_user"}); got.ID != permissionID || got.Name != "view_user" {
		t.Fatalf("ToPermissionResponse() = %+v", got)
	}
	refresh := &domain.RefreshToken{ID: tokenID, UserID: userID, TokenHash: "hashed-token", IpAddress: "192.0.2.1", UserAgent: "browser", ExpiresAt: created}
	refreshJSON, err := json.Marshal(ToRefreshTokenResponse(refresh))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(refreshJSON), "hashed-token") {
		t.Fatalf("ToRefreshTokenResponse() JSON=%s error=%v", refreshJSON, err)
	}
}

func TestPaginatedResponseMapsItemsAndNil(t *testing.T) {
	result := &shared.PaginatedResult[domain.User]{
		Items: []*domain.User{{Name: "Alice"}}, Total: 1, Page: 2, Limit: 5, TotalPages: 1,
	}
	got := ToPaginatedResponse(result, ToUserResponse)
	if len(got.Items) != 1 || got.Items[0].Name != "Alice" || got.Total != 1 || got.Page != 2 || got.Limit != 5 || got.TotalPages != 1 {
		t.Fatalf("ToPaginatedResponse() = %+v", got)
	}
	if empty := ToPaginatedResponse[domain.User, UserResponse](nil, ToUserResponse); empty.Items != nil || empty.Total != 0 {
		t.Fatalf("ToPaginatedResponse(nil) = %+v", empty)
	}
}
