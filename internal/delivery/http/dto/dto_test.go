package dto_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"uuid"

	commonDTO "gin-boilerplate/internal/delivery/http/dto"
	"gin-boilerplate/internal/domain/shared"
	permissiondto "gin-boilerplate/internal/permissions/adapters/http/dto"
	permissiondomain "gin-boilerplate/internal/permissions/domain"
	refreshtokendto "gin-boilerplate/internal/refresh_tokens/adapters/http/dto"
	refreshtokendomain "gin-boilerplate/internal/refresh_tokens/domain"
	roledto "gin-boilerplate/internal/roles/adapters/http/dto"
	roledomain "gin-boilerplate/internal/roles/domain"
	userdto "gin-boilerplate/internal/users/adapters/http/dto"
	userdomain "gin-boilerplate/internal/users/domain"
)

func TestEntityResponseMappersExposeOnlyPublicFields(t *testing.T) {
	userID, roleID, permissionID, tokenID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	created := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	user := &userdomain.User{
		ID:        userID,
		CreatedAt: created,
		UpdatedAt: created,
		Name:      "Alice",
		Email:     "alice@example.com",
		Password:  "secret-hash",
		Roles:     []roledomain.Role{{ID: roleID, Name: "Admin"}},
	}
	userResponse := userdto.ToUserResponse(user)
	if userResponse.ID != userID || userResponse.Name != "Alice" || userResponse.Email != "alice@example.com" || userResponse.CreatedAt != created {
		t.Fatalf("userdto.ToUserResponse() = %+v", userResponse)
	}
	userWithRole := userdto.ToUserWithRoleResponse(user)
	if userWithRole.ID != userID || len(userWithRole.Roles) != 1 || userWithRole.Roles[0].ID != roleID {
		t.Fatalf("userdto.ToUserWithRoleResponse() = %+v", userWithRole)
	}
	encoded, err := json.Marshal(userWithRole)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "secret-hash") {
		t.Fatalf("user response exposed password: %s", encoded)
	}

	role := &roledomain.Role{ID: roleID, Name: "Editor", Permissions: []permissiondomain.Permission{{ID: permissionID, Name: "view_user"}}}
	if got := roledto.ToRoleResponse(role); got.ID != roleID || got.Name != "Editor" {
		t.Fatalf("roledto.ToRoleResponse() = %+v", got)
	}
	if got := roledto.ToRoleWithPermissionsResponse(role); len(got.Permissions) != 1 || got.Permissions[0].ID != permissionID || got.Permissions[0].Name != "view_user" {
		t.Fatalf("roledto.ToRoleWithPermissionsResponse() = %+v", got)
	}
	if got := permissiondto.ToPermissionResponse(&permissiondomain.Permission{ID: permissionID, Name: "view_user"}); got.ID != permissionID || got.Name != "view_user" {
		t.Fatalf("permissiondto.ToPermissionResponse() = %+v", got)
	}
	refresh := &refreshtokendomain.RefreshToken{ID: tokenID, UserID: userID, TokenHash: "hashed-token", IpAddress: "192.0.2.1", UserAgent: "browser", ExpiresAt: created}
	refreshJSON, err := json.Marshal(refreshtokendto.ToRefreshTokenResponse(refresh))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(refreshJSON), "hashed-token") {
		t.Fatalf("refreshtokendto.ToRefreshTokenResponse() JSON=%s error=%v", refreshJSON, err)
	}
}

func TestPaginatedResponseMapsItemsAndNil(t *testing.T) {
	result := &shared.PaginatedResult[userdomain.User]{
		Items: []*userdomain.User{{Name: "Alice"}}, Total: 1, Page: 2, Limit: 5, TotalPages: 1,
	}
	got := commonDTO.ToPaginatedResponse(result, userdto.ToUserResponse)
	if len(got.Items) != 1 || got.Items[0].Name != "Alice" || got.Total != 1 || got.Page != 2 || got.Limit != 5 || got.TotalPages != 1 {
		t.Fatalf("commonDTO.ToPaginatedResponse() = %+v", got)
	}
	if empty := commonDTO.ToPaginatedResponse[userdomain.User, userdto.UserResponse](nil, userdto.ToUserResponse); empty.Items != nil || empty.Total != 0 {
		t.Fatalf("commonDTO.ToPaginatedResponse(nil) = %+v", empty)
	}
}
