package http

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	"gin-boilerplate/internal/delivery/http/middleware"
	"gin-boilerplate/internal/domain/shared"
	permissionhttp "gin-boilerplate/internal/permissions/adapters/http"
	permissiondomain "gin-boilerplate/internal/permissions/domain"
	rolehttp "gin-boilerplate/internal/roles/adapters/http"
	roledomain "gin-boilerplate/internal/roles/domain"
	userhttp "gin-boilerplate/internal/users/adapters/http"
	userdomain "gin-boilerplate/internal/users/domain"

	"github.com/gin-gonic/gin"
)

type httpUserUseCase struct {
	userdomain.UserUseCase
	user           *userdomain.User
	findErr        error
	listErr        error
	profileErr     error
	deleteErr      error
	addRolesErr    error
	removeRolesErr error
	imageErr       error
	removeImageErr error
	getImageErr    error
	listResult     *shared.PaginatedResult[userdomain.User]
	listParams     shared.PaginationParams
	listFilters    []shared.Filter
	findID         uuid.UUID
	updatedUser    *userdomain.User
	deletedID      uuid.UUID
	rolesUserID    uuid.UUID
	roleIDs        []uuid.UUID
	removedUserID  uuid.UUID
	removedRoleIDs []uuid.UUID
	imageUserID    uuid.UUID
	imageData      []byte
	removeImageID  uuid.UUID
	getImageID     uuid.UUID
	image          shared.ImageStream
	listCalls      int
}

func (f *httpUserUseCase) Find(_ context.Context, id uuid.UUID) (*userdomain.User, error) {
	f.findID = id
	return f.user, f.findErr
}

func (f *httpUserUseCase) List(_ context.Context, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[userdomain.User], error) {
	f.listCalls++
	f.listParams, f.listFilters = params, filters
	return f.listResult, f.listErr
}

func (f *httpUserUseCase) UpdateProfile(_ context.Context, user *userdomain.User) error {
	f.updatedUser = user
	if f.profileErr == nil && f.user != nil {
		user.Email = f.user.Email
	}
	return f.profileErr
}

func (f *httpUserUseCase) Delete(_ context.Context, id uuid.UUID) error {
	f.deletedID = id
	return f.deleteErr
}

func (f *httpUserUseCase) AddRoles(_ context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	f.rolesUserID, f.roleIDs = userID, append([]uuid.UUID(nil), roleIDs...)
	return f.addRolesErr
}

func (f *httpUserUseCase) RemoveRoles(_ context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	f.removedUserID, f.removedRoleIDs = userID, append([]uuid.UUID(nil), roleIDs...)
	return f.removeRolesErr
}

func (f *httpUserUseCase) UpdateImage(_ context.Context, userID uuid.UUID, file io.Reader) error {
	f.imageUserID = userID
	f.imageData, _ = io.ReadAll(file)
	return f.imageErr
}

func (f *httpUserUseCase) RemoveImage(_ context.Context, userID uuid.UUID) error {
	f.removeImageID = userID
	return f.removeImageErr
}

func (f *httpUserUseCase) GetImage(_ context.Context, userID uuid.UUID) (shared.ImageStream, error) {
	f.getImageID = userID
	return f.image, f.getImageErr
}

type httpRoleUseCase struct {
	roledomain.RoleUseCase
	role                 *roledomain.Role
	findErr              error
	listErr              error
	createErr            error
	updateErr            error
	deleteErr            error
	addPermissionsErr    error
	removePermissionsErr error
	listResult           *shared.PaginatedResult[roledomain.Role]
	listParams           shared.PaginationParams
	listFilters          []shared.Filter
	findID               uuid.UUID
	created              *roledomain.Role
	updated              *roledomain.Role
	deletedID            uuid.UUID
	permissionRoleID     uuid.UUID
	permissionIDs        []uuid.UUID
	removedRoleID        uuid.UUID
	removedPermissionIDs []uuid.UUID
	listCalls            int
}

func (f *httpRoleUseCase) Find(_ context.Context, id uuid.UUID) (*roledomain.Role, error) {
	f.findID = id
	return f.role, f.findErr
}

func (f *httpRoleUseCase) List(_ context.Context, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[roledomain.Role], error) {
	f.listCalls++
	f.listParams, f.listFilters = params, filters
	return f.listResult, f.listErr
}

func (f *httpRoleUseCase) Create(_ context.Context, role *roledomain.Role) error {
	f.created = role
	return f.createErr
}

func (f *httpRoleUseCase) Update(_ context.Context, role *roledomain.Role) error {
	f.updated = role
	return f.updateErr
}

func (f *httpRoleUseCase) Delete(_ context.Context, id uuid.UUID) error {
	f.deletedID = id
	return f.deleteErr
}

func (f *httpRoleUseCase) AddPermissions(_ context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	f.permissionRoleID, f.permissionIDs = roleID, append([]uuid.UUID(nil), permissionIDs...)
	return f.addPermissionsErr
}

func (f *httpRoleUseCase) RemovePermissions(_ context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	f.removedRoleID, f.removedPermissionIDs = roleID, append([]uuid.UUID(nil), permissionIDs...)
	return f.removePermissionsErr
}

type httpPermissionUseCase struct {
	permissiondomain.PermissionUseCase
	result  *shared.PaginatedResult[permissiondomain.Permission]
	params  shared.PaginationParams
	filters []shared.Filter
	err     error
	calls   int
}

type httpActivityLogUseCase struct {
	activitylogdomain.ActivityLogUseCase
	result  *shared.PaginatedResult[activitylogdomain.ActivityLog]
	params  shared.PaginationParams
	filters []shared.Filter
	err     error
	calls   int
}

func (f *httpActivityLogUseCase) List(_ context.Context, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[activitylogdomain.ActivityLog], error) {
	f.calls++
	f.params, f.filters = params, filters
	return f.result, f.err
}

func (f *httpPermissionUseCase) List(_ context.Context, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[permissiondomain.Permission], error) {
	f.calls++
	f.params, f.filters = params, filters
	return f.result, f.err
}

func TestUserProfileAndListRoutesForwardIDsAndFilters(t *testing.T) {
	userID := uuid.New()
	fake := &httpUserUseCase{
		user:       &userdomain.User{ID: userID, Name: "Alice", Email: "alice@example.com", Password: "must-not-be-returned"},
		listResult: &shared.PaginatedResult[userdomain.User]{Page: 2, Limit: 5, Total: 1, Items: []*userdomain.User{{ID: userID, Name: "Alice", Email: "alice@example.com"}}},
	}
	handler := userhttp.NewUserHandler(fake)

	profile := serveHTTPHandler(t, http.MethodGet, "/profile", "/profile", "", handler.GetProfile, withAuthenticatedUser(userID))
	if profile.Code != http.StatusOK || fake.findID != userID || decodeResponse(t, profile).Success != true {
		t.Fatalf("GetProfile() status=%d id=%v body=%s", profile.Code, fake.findID, profile.Body.String())
	}
	if strings.Contains(profile.Body.String(), "must-not-be-returned") || strings.Contains(profile.Body.String(), "password") {
		t.Fatalf("profile response exposed a password field: %s", profile.Body.String())
	}
	updated := serveHTTPHandler(t, http.MethodPut, "/profile", "/profile", `{"name":"Alice Updated"}`, handler.UpdateProfile, withAuthenticatedUser(userID))
	if updated.Code != http.StatusOK || fake.updatedUser == nil || fake.updatedUser.ID != userID || fake.updatedUser.Name != "Alice Updated" {
		t.Fatalf("UpdateProfile() status=%d user=%+v body=%s", updated.Code, fake.updatedUser, updated.Body.String())
	}

	list := serveHTTPHandler(t, http.MethodGet, "/users", "/users?page=2&limit=5&sort=name:desc&name=Ali&email=example.com", "", handler.ListUsers)
	if list.Code != http.StatusOK || fake.listParams.Page != 2 || fake.listParams.Limit != 5 || len(fake.listParams.Sort) != 1 || fake.listParams.Sort[0].Direction != shared.SortDesc || len(fake.listFilters) != 2 {
		t.Fatalf("ListUsers() status=%d params=%+v filters=%+v", list.Code, fake.listParams, fake.listFilters)
	}
	if fake.listFilters[0].Value != "%Ali%" || fake.listFilters[1].Value != "%example.com%" {
		t.Fatalf("ListUsers() filters = %+v", fake.listFilters)
	}
	invalid := serveHTTPHandler(t, http.MethodGet, "/users", "/users?page=0", "", handler.ListUsers)
	if invalid.Code != http.StatusUnprocessableEntity || fake.listCalls != 1 {
		t.Fatalf("ListUsers() invalid pagination status=%d calls=%d", invalid.Code, fake.listCalls)
	}
}

func TestUserFindAndAdminUpdateRoutesUsePathID(t *testing.T) {
	userID := uuid.New()
	fake := &httpUserUseCase{user: &userdomain.User{ID: userID, Name: "Alice", Email: "alice@example.com"}}
	handler := userhttp.NewUserHandler(fake)
	find := serveHTTPHandler(t, http.MethodGet, "/users/:id", "/users/"+userID.String(), "", handler.FindUser)
	if find.Code != http.StatusOK || fake.findID != userID {
		t.Fatalf("FindUser() status=%d ID=%v body=%s", find.Code, fake.findID, find.Body.String())
	}
	update := serveHTTPHandler(t, http.MethodPut, "/users/:id", "/users/"+userID.String(), `{"name":"Admin Update"}`, handler.UpdateUser)
	if update.Code != http.StatusOK || fake.updatedUser == nil || fake.updatedUser.ID != userID || fake.updatedUser.Name != "Admin Update" {
		t.Fatalf("UpdateUser() status=%d user=%+v body=%s", update.Code, fake.updatedUser, update.Body.String())
	}
}

func TestUserReadRoutesRejectInvalidIdentityAndReturnNotFound(t *testing.T) {
	userID := uuid.New()
	fake := &httpUserUseCase{findErr: shared.NewAppError(shared.ErrTypeNotFound, "user not found", nil)}
	handler := userhttp.NewUserHandler(fake)

	missingContext := serveHTTPHandler(t, http.MethodGet, "/profile", "/profile", "", handler.GetProfile)
	if missingContext.Code != http.StatusUnauthorized || fake.findID != uuid.Nil() {
		t.Fatalf("GetProfile() missing identity status=%d findID=%v", missingContext.Code, fake.findID)
	}
	invalidID := serveHTTPHandler(t, http.MethodGet, "/users/:id", "/users/not-a-uuid", "", handler.FindUser)
	if invalidID.Code != http.StatusUnprocessableEntity || fake.findID != uuid.Nil() {
		t.Fatalf("FindUser() invalid ID status=%d findID=%v", invalidID.Code, fake.findID)
	}
	notFound := serveHTTPHandler(t, http.MethodGet, "/users/:id", "/users/"+userID.String(), "", handler.FindUser)
	if notFound.Code != http.StatusNotFound || decodeResponse(t, notFound).Error != "user not found" || fake.findID != userID {
		t.Fatalf("FindUser() missing user status=%d findID=%v body=%s", notFound.Code, fake.findID, notFound.Body.String())
	}
}

func TestUserRoleAndDeleteRoutesBindPathAndJSON(t *testing.T) {
	userID, roleID := uuid.New(), uuid.New()
	fake := &httpUserUseCase{}
	handler := userhttp.NewUserHandler(fake)
	body := `{"role_ids":["` + roleID.String() + `"]}`

	added := serveHTTPHandler(t, http.MethodPost, "/users/:id/roles", "/users/"+userID.String()+"/roles", body, handler.AddRoles)
	if added.Code != http.StatusOK || fake.rolesUserID != userID || len(fake.roleIDs) != 1 || fake.roleIDs[0] != roleID {
		t.Fatalf("AddRoles() status=%d user=%v roles=%v", added.Code, fake.rolesUserID, fake.roleIDs)
	}
	removed := serveHTTPHandler(t, http.MethodDelete, "/users/:id/roles", "/users/"+userID.String()+"/roles", body, handler.RemoveRoles)
	if removed.Code != http.StatusNoContent || removed.Body.Len() != 0 || fake.removedUserID != userID || len(fake.removedRoleIDs) != 1 || fake.removedRoleIDs[0] != roleID {
		t.Fatalf("RemoveRoles() status=%d user=%v roles=%v body=%q", removed.Code, fake.removedUserID, fake.removedRoleIDs, removed.Body.String())
	}
	deleted := serveHTTPHandler(t, http.MethodDelete, "/users/:id", "/users/"+userID.String(), "", handler.DeleteUser)
	if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 || fake.deletedID != userID {
		t.Fatalf("DeleteUser() status=%d ID=%v body=%q", deleted.Code, fake.deletedID, deleted.Body.String())
	}
}

func TestRoleAndPermissionRoutes(t *testing.T) {
	roleID, permissionID := uuid.New(), uuid.New()
	roleFake := &httpRoleUseCase{role: &roledomain.Role{ID: roleID, Name: "Editor"}, listResult: &shared.PaginatedResult[roledomain.Role]{Page: 1}}
	roleHandler := rolehttp.NewRoleHandler(roleFake)
	created := serveHTTPHandler(t, http.MethodPost, "/roles", "/roles", `{"name":"Editor"}`, roleHandler.CreateRole)
	if created.Code != http.StatusCreated || roleFake.created == nil || roleFake.created.Name != "Editor" {
		t.Fatalf("CreateRole() status=%d role=%+v", created.Code, roleFake.created)
	}
	updated := serveHTTPHandler(t, http.MethodPut, "/roles/:id", "/roles/"+roleID.String(), `{"name":"Operator"}`, roleHandler.UpdateRole)
	if updated.Code != http.StatusOK || roleFake.updated == nil || roleFake.updated.ID != roleID || roleFake.updated.Name != "Operator" {
		t.Fatalf("UpdateRole() status=%d role=%+v", updated.Code, roleFake.updated)
	}
	listed := serveHTTPHandler(t, http.MethodGet, "/roles", "/roles?name=Edit&sort=-name", "", roleHandler.ListRoles)
	if listed.Code != http.StatusOK || roleFake.listParams.Sort[0].Direction != shared.SortDesc || len(roleFake.listFilters) != 1 || roleFake.listFilters[0].Value != "%Edit%" {
		t.Fatalf("ListRoles() status=%d params=%+v filters=%+v", listed.Code, roleFake.listParams, roleFake.listFilters)
	}
	body := `{"permission_ids":["` + permissionID.String() + `"]}`
	added := serveHTTPHandler(t, http.MethodPost, "/roles/:id/permissions", "/roles/"+roleID.String()+"/permissions", body, roleHandler.AddPermissions)
	if added.Code != http.StatusCreated || roleFake.permissionRoleID != roleID || len(roleFake.permissionIDs) != 1 || roleFake.permissionIDs[0] != permissionID {
		t.Fatalf("AddPermissions() status=%d role=%v permission IDs=%v", added.Code, roleFake.permissionRoleID, roleFake.permissionIDs)
	}
	removed := serveHTTPHandler(t, http.MethodDelete, "/roles/:id/permissions", "/roles/"+roleID.String()+"/permissions", body, roleHandler.RemovePermissions)
	if removed.Code != http.StatusNoContent || removed.Body.Len() != 0 || roleFake.removedRoleID != roleID || len(roleFake.removedPermissionIDs) != 1 {
		t.Fatalf("RemovePermissions() status=%d role=%v IDs=%v", removed.Code, roleFake.removedRoleID, roleFake.removedPermissionIDs)
	}
	deleted := serveHTTPHandler(t, http.MethodDelete, "/roles/:id", "/roles/"+roleID.String(), "", roleHandler.DeleteRole)
	if deleted.Code != http.StatusNoContent || roleFake.deletedID != roleID {
		t.Fatalf("DeleteRole() status=%d ID=%v", deleted.Code, roleFake.deletedID)
	}

	permissionFake := &httpPermissionUseCase{result: &shared.PaginatedResult[permissiondomain.Permission]{Page: 1}}
	permissionHandler := permissionhttp.NewPermissionHandler(permissionFake)
	permissions := serveHTTPHandler(t, http.MethodGet, "/permissions", "/permissions?name=role&limit=5", "", permissionHandler.ListPermissions)
	if permissions.Code != http.StatusOK || permissionFake.params.Limit != 5 || len(permissionFake.filters) != 1 || permissionFake.filters[0].Value != "%role%" {
		t.Fatalf("ListPermissions() status=%d params=%+v filters=%+v", permissions.Code, permissionFake.params, permissionFake.filters)
	}
}

func TestFindRoleReturnsPermissions(t *testing.T) {
	roleID, permissionID := uuid.New(), uuid.New()
	fake := &httpRoleUseCase{role: &roledomain.Role{
		ID:          roleID,
		Name:        "Editor",
		Permissions: []permissiondomain.Permission{{ID: permissionID, Name: "view_user"}},
	}}
	handler := rolehttp.NewRoleHandler(fake)
	recorder := serveHTTPHandler(t, http.MethodGet, "/roles/:id", "/roles/"+roleID.String(), "", handler.FindRole)
	if recorder.Code != http.StatusOK || fake.findID != roleID {
		t.Fatalf("FindRole() status=%d ID=%v body=%s", recorder.Code, fake.findID, recorder.Body.String())
	}
	data, ok := decodeResponse(t, recorder).Data.(map[string]any)
	permissions, permissionsOK := data["permissions"].([]any)
	if !ok || !permissionsOK || len(permissions) != 1 {
		t.Fatalf("FindRole() response data=%v", data)
	}
}

func TestUserHandlersReturnUseCaseErrors(t *testing.T) {
	userID, roleID := uuid.New(), uuid.New()
	notFound := shared.NewAppError(shared.ErrTypeNotFound, "resource missing", nil)
	validRoles := `{"role_ids":["` + roleID.String() + `"]}`
	cases := []struct {
		name string
		run  func(*httpUserUseCase, *userhttp.UserHandler) *httptest.ResponseRecorder
	}{
		{name: "profile lookup", run: func(fake *httpUserUseCase, handler *userhttp.UserHandler) *httptest.ResponseRecorder {
			fake.findErr = notFound
			return serveHTTPHandler(t, http.MethodGet, "/profile", "/profile", "", handler.GetProfile, withAuthenticatedUser(userID))
		}},
		{name: "profile update", run: func(fake *httpUserUseCase, handler *userhttp.UserHandler) *httptest.ResponseRecorder {
			fake.profileErr = notFound
			return serveHTTPHandler(t, http.MethodPut, "/profile", "/profile", `{"name":"Updated"}`, handler.UpdateProfile, withAuthenticatedUser(userID))
		}},
		{name: "admin update", run: func(fake *httpUserUseCase, handler *userhttp.UserHandler) *httptest.ResponseRecorder {
			fake.profileErr = notFound
			return serveHTTPHandler(t, http.MethodPut, "/users/:id", "/users/"+userID.String(), `{"name":"Updated"}`, handler.UpdateUser)
		}},
		{name: "delete", run: func(fake *httpUserUseCase, handler *userhttp.UserHandler) *httptest.ResponseRecorder {
			fake.deleteErr = notFound
			return serveHTTPHandler(t, http.MethodDelete, "/users/:id", "/users/"+userID.String(), "", handler.DeleteUser)
		}},
		{name: "add roles", run: func(fake *httpUserUseCase, handler *userhttp.UserHandler) *httptest.ResponseRecorder {
			fake.addRolesErr = notFound
			return serveHTTPHandler(t, http.MethodPost, "/users/:id/roles", "/users/"+userID.String()+"/roles", validRoles, handler.AddRoles)
		}},
		{name: "remove roles", run: func(fake *httpUserUseCase, handler *userhttp.UserHandler) *httptest.ResponseRecorder {
			fake.removeRolesErr = notFound
			return serveHTTPHandler(t, http.MethodDelete, "/users/:id/roles", "/users/"+userID.String()+"/roles", validRoles, handler.RemoveRoles)
		}},
		{name: "remove image", run: func(fake *httpUserUseCase, handler *userhttp.UserHandler) *httptest.ResponseRecorder {
			fake.removeImageErr = notFound
			return serveHTTPHandler(t, http.MethodDelete, "/users/profile/image", "/users/profile/image", "", handler.RemoveImage, withAuthenticatedUser(userID))
		}},
		{name: "get image", run: func(fake *httpUserUseCase, handler *userhttp.UserHandler) *httptest.ResponseRecorder {
			fake.getImageErr = notFound
			return serveHTTPHandler(t, http.MethodGet, "/users/:id/image", "/users/"+userID.String()+"/image", "", handler.GetImage)
		}},
		{name: "list", run: func(fake *httpUserUseCase, handler *userhttp.UserHandler) *httptest.ResponseRecorder {
			fake.listErr = notFound
			return serveHTTPHandler(t, http.MethodGet, "/users", "/users", "", handler.ListUsers)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &httpUserUseCase{}
			recorder := tc.run(fake, userhttp.NewUserHandler(fake))
			if recorder.Code != http.StatusNotFound || decodeResponse(t, recorder).Error != "resource missing" {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestRoleAndPermissionHandlersReturnUseCaseErrors(t *testing.T) {
	roleID, permissionID := uuid.New(), uuid.New()
	notFound := shared.NewAppError(shared.ErrTypeNotFound, "resource missing", nil)
	validPermissions := `{"permission_ids":["` + permissionID.String() + `"]}`
	cases := []struct {
		name string
		run  func(*httpRoleUseCase, *rolehttp.RoleHandler) *httptest.ResponseRecorder
	}{
		{name: "find", run: func(fake *httpRoleUseCase, handler *rolehttp.RoleHandler) *httptest.ResponseRecorder {
			fake.findErr = notFound
			return serveHTTPHandler(t, http.MethodGet, "/roles/:id", "/roles/"+roleID.String(), "", handler.FindRole)
		}},
		{name: "list", run: func(fake *httpRoleUseCase, handler *rolehttp.RoleHandler) *httptest.ResponseRecorder {
			fake.listErr = notFound
			return serveHTTPHandler(t, http.MethodGet, "/roles", "/roles", "", handler.ListRoles)
		}},
		{name: "create", run: func(fake *httpRoleUseCase, handler *rolehttp.RoleHandler) *httptest.ResponseRecorder {
			fake.createErr = notFound
			return serveHTTPHandler(t, http.MethodPost, "/roles", "/roles", `{"name":"Editor"}`, handler.CreateRole)
		}},
		{name: "update", run: func(fake *httpRoleUseCase, handler *rolehttp.RoleHandler) *httptest.ResponseRecorder {
			fake.updateErr = notFound
			return serveHTTPHandler(t, http.MethodPut, "/roles/:id", "/roles/"+roleID.String(), `{"name":"Editor"}`, handler.UpdateRole)
		}},
		{name: "delete", run: func(fake *httpRoleUseCase, handler *rolehttp.RoleHandler) *httptest.ResponseRecorder {
			fake.deleteErr = notFound
			return serveHTTPHandler(t, http.MethodDelete, "/roles/:id", "/roles/"+roleID.String(), "", handler.DeleteRole)
		}},
		{name: "add permissions", run: func(fake *httpRoleUseCase, handler *rolehttp.RoleHandler) *httptest.ResponseRecorder {
			fake.addPermissionsErr = notFound
			return serveHTTPHandler(t, http.MethodPost, "/roles/:id/permissions", "/roles/"+roleID.String()+"/permissions", validPermissions, handler.AddPermissions)
		}},
		{name: "remove permissions", run: func(fake *httpRoleUseCase, handler *rolehttp.RoleHandler) *httptest.ResponseRecorder {
			fake.removePermissionsErr = notFound
			return serveHTTPHandler(t, http.MethodDelete, "/roles/:id/permissions", "/roles/"+roleID.String()+"/permissions", validPermissions, handler.RemovePermissions)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &httpRoleUseCase{}
			recorder := tc.run(fake, rolehttp.NewRoleHandler(fake))
			if recorder.Code != http.StatusNotFound || decodeResponse(t, recorder).Error != "resource missing" {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}

	permissionFake := &httpPermissionUseCase{err: notFound}
	permissionRecorder := serveHTTPHandler(t, http.MethodGet, "/permissions", "/permissions", "", permissionhttp.NewPermissionHandler(permissionFake).ListPermissions)
	if permissionRecorder.Code != http.StatusNotFound || decodeResponse(t, permissionRecorder).Error != "resource missing" {
		t.Fatalf("permission list status=%d body=%s", permissionRecorder.Code, permissionRecorder.Body.String())
	}
}

type trackedReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackedReadCloser) Close() error { r.closed = true; return nil }

func TestUserImageRoutesUploadAndStreamContent(t *testing.T) {
	userID := uuid.New()
	stream := &trackedReadCloser{Reader: strings.NewReader("image-bytes")}
	fake := &httpUserUseCase{image: shared.ImageStream{Content: stream, ContentType: "image/png", Size: int64(len("image-bytes"))}}
	handler := userhttp.NewUserHandler(fake)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("image", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("uploaded-image")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	router.PUT("/users/profile/image", func(c *gin.Context) {
		c.Set("user_id", userID.String())
		handler.UpdateImage(c)
	})
	req := httptest.NewRequest(http.MethodPut, "/users/profile/image", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK || fake.imageUserID != userID || string(fake.imageData) != "uploaded-image" {
		t.Fatalf("UpdateImage() status=%d user=%v data=%q", recorder.Code, fake.imageUserID, fake.imageData)
	}

	image := serveHTTPHandler(t, http.MethodGet, "/users/:id/image", "/users/"+userID.String()+"/image", "", handler.GetImage)
	if image.Code != http.StatusOK || image.Header().Get("Content-Type") != "image/png" || image.Body.String() != "image-bytes" || fake.getImageID != userID || !stream.closed {
		t.Fatalf("GetImage() status=%d type=%q body=%q user=%v closed=%v", image.Code, image.Header().Get("Content-Type"), image.Body.String(), fake.getImageID, stream.closed)
	}
	removed := serveHTTPHandler(t, http.MethodDelete, "/users/profile/image", "/users/profile/image", "", handler.RemoveImage, withAuthenticatedUser(userID))
	if removed.Code != http.StatusNoContent || removed.Body.Len() != 0 || fake.removeImageID != userID {
		t.Fatalf("RemoveImage() status=%d user=%v body=%q", removed.Code, fake.removeImageID, removed.Body.String())
	}
}

func TestUserImageUploadRequiresFileAndEnforcesRequestLimit(t *testing.T) {
	userID := uuid.New()
	handler := userhttp.NewUserHandler(&httpUserUseCase{})
	makeRequest := func(fileContent []byte, includeFile bool) *httptest.ResponseRecorder {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if includeFile {
			file, err := writer.CreateFormFile("image", "avatar.png")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.Write(fileContent); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(middleware.ErrorHandler())
		router.PUT("/image", func(c *gin.Context) {
			c.Set("user_id", userID.String())
			handler.UpdateImage(c)
		})
		req := httptest.NewRequest(http.MethodPut, "/image", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		return recorder
	}

	missing := makeRequest(nil, false)
	if missing.Code != http.StatusUnprocessableEntity || decodeResponse(t, missing).Error != "File is required" {
		t.Fatalf("missing image status=%d body=%s", missing.Code, missing.Body.String())
	}
	oversized := makeRequest(bytes.Repeat([]byte("x"), 5*1024*1024), true)
	if oversized.Code != http.StatusUnprocessableEntity || decodeResponse(t, oversized).Error != "Upload request is too large" {
		t.Fatalf("oversized image status=%d body=%s", oversized.Code, oversized.Body.String())
	}
}
