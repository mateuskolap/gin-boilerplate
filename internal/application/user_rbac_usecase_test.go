package application_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
	"uuid"

	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/domain/shared"
	permissiondomain "gin-boilerplate/internal/permissions/domain"
	roleapp "gin-boilerplate/internal/roles/application"
	roledomain "gin-boilerplate/internal/roles/domain"
	userapp "gin-boilerplate/internal/users/application"
	userdomain "gin-boilerplate/internal/users/domain"
)

type rbacUserRepo struct {
	userdomain.UserRepository
	users          map[uuid.UUID]*userdomain.User
	getErr         error
	updateErr      error
	deleteErr      error
	addRolesErr    error
	removeRolesErr error
	listErr        error
	listResult     *shared.PaginatedResult[userdomain.User]
	updated        *userdomain.User
	deletedID      uuid.UUID
	addedUser      userdomain.User
	addedRoleIDs   []uuid.UUID
	removedUser    userdomain.User
	removedRoleIDs []uuid.UUID
	listParams     shared.PaginationParams
	listFilters    []shared.Filter
	listCalls      int
	loadedRoles    bool
}

func (r *rbacUserRepo) GetByID(_ context.Context, id uuid.UUID) (*userdomain.User, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.users[id], nil
}

func (r *rbacUserRepo) GetByIDWithRoles(ctx context.Context, id uuid.UUID) (*userdomain.User, error) {
	r.loadedRoles = true
	return r.GetByID(ctx, id)
}

func (r *rbacUserRepo) Update(_ context.Context, user *userdomain.User) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updated = user
	r.users[user.ID] = user
	return nil
}

func (r *rbacUserRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.deletedID = id
	return r.deleteErr
}

func (r *rbacUserRepo) AddRoles(_ context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	r.addedUser.ID, r.addedRoleIDs = userID, append([]uuid.UUID(nil), roleIDs...)
	return r.addRolesErr
}

func (r *rbacUserRepo) RemoveRoles(_ context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	r.removedUser.ID, r.removedRoleIDs = userID, append([]uuid.UUID(nil), roleIDs...)
	return r.removeRolesErr
}

func (r *rbacUserRepo) List(_ context.Context, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[userdomain.User], error) {
	r.listCalls++
	r.listParams, r.listFilters = params, filters
	return r.listResult, r.listErr
}

type rbacRoleRepo struct {
	roledomain.RoleRepository
	roles                map[uuid.UUID]*roledomain.Role
	byName               *roledomain.Role
	getByNameErr         error
	getErr               error
	createErr            error
	updateErr            error
	deleteErr            error
	addPermissionsErr    error
	removePermissionsErr error
	listErr              error
	listResult           *shared.PaginatedResult[roledomain.Role]
	queriedName          string
	created              *roledomain.Role
	updated              *roledomain.Role
	addedRole            roledomain.Role
	addedPermissionIDs   []uuid.UUID
	removedRole          roledomain.Role
	removedPermissionIDs []uuid.UUID
	deletedID            uuid.UUID
	listParams           shared.PaginationParams
	listFilters          []shared.Filter
	listCalls            int
	loadedPermissions    bool
}

func (r *rbacRoleRepo) GetByName(_ context.Context, name string) (*roledomain.Role, error) {
	r.queriedName = name
	return r.byName, r.getByNameErr
}

func (r *rbacRoleRepo) GetByID(_ context.Context, id uuid.UUID) (*roledomain.Role, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.roles[id], nil
}

func (r *rbacRoleRepo) GetByIDWithPermissions(ctx context.Context, id uuid.UUID) (*roledomain.Role, error) {
	r.loadedPermissions = true
	return r.GetByID(ctx, id)
}

func (r *rbacRoleRepo) Create(_ context.Context, role *roledomain.Role) error {
	r.created = role
	return r.createErr
}

func (r *rbacRoleRepo) Update(_ context.Context, role *roledomain.Role) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updated = role
	r.roles[role.ID] = role
	return nil
}

func (r *rbacRoleRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.deletedID = id
	return r.deleteErr
}

func (r *rbacRoleRepo) AddPermissions(_ context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	r.addedRole.ID, r.addedPermissionIDs = roleID, append([]uuid.UUID(nil), permissionIDs...)
	return r.addPermissionsErr
}

func (r *rbacRoleRepo) RemovePermissions(_ context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	r.removedRole.ID, r.removedPermissionIDs = roleID, append([]uuid.UUID(nil), permissionIDs...)
	return r.removePermissionsErr
}

func (r *rbacRoleRepo) List(_ context.Context, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[roledomain.Role], error) {
	r.listCalls++
	r.listParams, r.listFilters = params, filters
	return r.listResult, r.listErr
}

type userTestStorage struct {
	putErr      error
	openErr     error
	deleteErr   error
	putKey      string
	putOptions  port.PutOptions
	putContent  []byte
	openKey     string
	openResult  port.OpenedFile
	deletedKeys []string
}

func (s *userTestStorage) Put(_ context.Context, key string, src io.Reader, options port.PutOptions) error {
	s.putKey, s.putOptions = key, options
	if s.putErr != nil {
		return s.putErr
	}
	content, err := io.ReadAll(src)
	s.putContent = content
	return err
}

func (s *userTestStorage) Open(_ context.Context, key string) (port.OpenedFile, error) {
	s.openKey = key
	return s.openResult, s.openErr
}

func (s *userTestStorage) Delete(_ context.Context, key string) error {
	s.deletedKeys = append(s.deletedKeys, key)
	return s.deleteErr
}

type userTestImageInspector struct {
	image port.InspectedImage
	err   error
}

func (i *userTestImageInspector) Inspect(context.Context, io.Reader) (port.InspectedImage, error) {
	return i.image, i.err
}

func newUserUseCaseForTest(repo *rbacUserRepo, refresh *testRefreshUseCase, storage port.Storage, inspector port.ImageInspector, blacklist *testBlacklist, tx *testTransaction) userdomain.UserUseCase {
	return userapp.NewUserUseCase(repo, refresh, storage, inspector, blacklist, tx, time.Hour, &testActivityLogRepo{})
}

func TestUserUpdateProfileTrimsNameAndValidatesRunes(t *testing.T) {
	id := uuid.New()
	repo := &rbacUserRepo{users: map[uuid.UUID]*userdomain.User{id: {ID: id, Name: "Old name"}}}
	useCase := newUserUseCaseForTest(repo, &testRefreshUseCase{}, &userTestStorage{}, &userTestImageInspector{}, &testBlacklist{}, &testTransaction{})
	user := &userdomain.User{ID: id, Name: "  New name  "}

	if err := useCase.UpdateProfile(context.Background(), user); err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if user.Name != "New name" || repo.updated == nil || repo.updated.ID != id || repo.updated.Name != "New name" {
		t.Fatalf("UpdateProfile() input=%+v persisted=%+v", user, repo.updated)
	}
	if err := useCase.UpdateProfile(context.Background(), &userdomain.User{ID: id, Name: "é"}); appErrorType(err) != shared.ErrTypeValidation {
		t.Fatalf("UpdateProfile() accepted a one-rune name: %v", err)
	}
}

func TestUserRoleChangesInvalidateAccessTokens(t *testing.T) {
	userID, roleID := uuid.New(), uuid.New()
	roleIDs := []uuid.UUID{roleID}
	for _, tc := range []struct {
		name string
		call func(userdomain.UserUseCase) error
	}{
		{name: "add", call: func(uc userdomain.UserUseCase) error { return uc.AddRoles(context.Background(), userID, roleIDs) }},
		{name: "remove", call: func(uc userdomain.UserUseCase) error { return uc.RemoveRoles(context.Background(), userID, roleIDs) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := &userdomain.User{ID: userID}
			if tc.name == "remove" {
				user.Roles = []roledomain.Role{{ID: roleID}}
			}
			repo := &rbacUserRepo{users: map[uuid.UUID]*userdomain.User{userID: user}}
			blacklist := &testBlacklist{}
			uc := newUserUseCaseForTest(repo, &testRefreshUseCase{}, &userTestStorage{}, &userTestImageInspector{}, blacklist, &testTransaction{})
			if err := tc.call(uc); err != nil {
				t.Fatalf("role change error = %v", err)
			}
			if blacklist.revokedUserID != userID.String() || blacklist.revokeUserTTL != time.Hour {
				t.Fatalf("role change did not invalidate user access tokens: %+v", blacklist)
			}
			if tc.name == "add" && (repo.addedUser.ID != userID || len(repo.addedRoleIDs) != 1 || repo.addedRoleIDs[0] != roleID) {
				t.Fatalf("AddRoles() passed wrong user or role IDs: %+v %v", repo.addedUser, repo.addedRoleIDs)
			}
			if tc.name == "remove" && (repo.removedUser.ID != userID || len(repo.removedRoleIDs) != 1 || repo.removedRoleIDs[0] != roleID) {
				t.Fatalf("RemoveRoles() passed wrong user or role IDs: %+v %v", repo.removedUser, repo.removedRoleIDs)
			}
		})
	}
}

func TestUserRoleChangeFailuresDoNotHidePartialFailure(t *testing.T) {
	userID, roleID := uuid.New(), uuid.New()
	user := &userdomain.User{ID: userID}

	t.Run("repository failure stops before token invalidation", func(t *testing.T) {
		repo := &rbacUserRepo{users: map[uuid.UUID]*userdomain.User{userID: user}, addRolesErr: io.ErrClosedPipe}
		blacklist := &testBlacklist{}
		useCase := newUserUseCaseForTest(repo, &testRefreshUseCase{}, &userTestStorage{}, &userTestImageInspector{}, blacklist, &testTransaction{})
		if err := useCase.AddRoles(context.Background(), userID, []uuid.UUID{roleID}); appErrorType(err) != shared.ErrTypeInternal {
			t.Fatalf("AddRoles() error=%v, want internal", err)
		}
		if blacklist.revokedUserID != "" {
			t.Fatalf("AddRoles() invalidated tokens although the role update failed: %+v", blacklist)
		}
	})

	t.Run("token invalidation failure is returned", func(t *testing.T) {
		user := &userdomain.User{ID: userID, Roles: []roledomain.Role{{ID: roleID}}}
		repo := &rbacUserRepo{users: map[uuid.UUID]*userdomain.User{userID: user}}
		blacklist := &testBlacklist{revokeUserErr: io.ErrClosedPipe}
		useCase := newUserUseCaseForTest(repo, &testRefreshUseCase{}, &userTestStorage{}, &userTestImageInspector{}, blacklist, &testTransaction{})
		if err := useCase.RemoveRoles(context.Background(), userID, []uuid.UUID{roleID}); appErrorType(err) != shared.ErrTypeInternal {
			t.Fatalf("RemoveRoles() error=%v, want internal", err)
		}
		if repo.removedUser.ID != userID || blacklist.revokedUserID != userID.String() {
			t.Fatalf("RemoveRoles() did not perform the role update and attempt token invalidation: repo=%+v blacklist=%+v", repo, blacklist)
		}
	})
}

func TestUserDeleteRevokesSessionsAndCleansAvatar(t *testing.T) {
	userID := uuid.New()
	user := &userdomain.User{ID: userID, AvatarKey: "users/avatar.jpg"}
	repo := &rbacUserRepo{users: map[uuid.UUID]*userdomain.User{userID: user}}
	refresh, blacklist, tx, storage := &testRefreshUseCase{}, &testBlacklist{}, &testTransaction{}, &userTestStorage{}
	useCase := newUserUseCaseForTest(repo, refresh, storage, &userTestImageInspector{}, blacklist, tx)

	if err := useCase.Delete(context.Background(), userID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if tx.calls != 1 || refresh.revokedAllUserID != userID || repo.deletedID != userID {
		t.Fatalf("Delete() failed to delete user and refresh sessions: tx=%d refresh=%+v deleted=%v", tx.calls, refresh, repo.deletedID)
	}
	if len(storage.deletedKeys) != 1 || storage.deletedKeys[0] != user.AvatarKey || blacklist.revokedUserID != userID.String() {
		t.Fatalf("Delete() failed to clean avatar or invalidate access tokens: storage=%v blacklist=%+v", storage.deletedKeys, blacklist)
	}
}

func TestUserDeleteAbortsWhenRefreshSessionsCannotBeRevoked(t *testing.T) {
	userID := uuid.New()
	user := &userdomain.User{ID: userID, AvatarKey: "users/avatar.jpg"}
	repo := &rbacUserRepo{users: map[uuid.UUID]*userdomain.User{userID: user}}
	refresh := &testRefreshUseCase{revokeAllErr: io.ErrClosedPipe}
	blacklist, tx, storage := &testBlacklist{}, &testTransaction{}, &userTestStorage{}
	useCase := newUserUseCaseForTest(repo, refresh, storage, &userTestImageInspector{}, blacklist, tx)

	if err := useCase.Delete(context.Background(), userID); appErrorType(err) != shared.ErrTypeInternal {
		t.Fatalf("Delete() error=%v, want internal", err)
	}
	if tx.calls != 1 || repo.deletedID != uuid.Nil() || len(storage.deletedKeys) != 0 || blacklist.revokedUserID != "" {
		t.Fatalf("Delete() continued after refresh revocation failure: tx=%d deleted=%v storage=%v blacklist=%+v", tx.calls, repo.deletedID, storage.deletedKeys, blacklist)
	}
}

func TestUserAvatarUpdateReplacesAndCleansFiles(t *testing.T) {
	userID := uuid.New()
	user := &userdomain.User{ID: userID, AvatarKey: "users/old.jpg"}
	repo := &rbacUserRepo{users: map[uuid.UUID]*userdomain.User{userID: user}}
	storage := &userTestStorage{}
	inspector := &userTestImageInspector{image: port.InspectedImage{Content: strings.NewReader("png-data"), Extension: ".png"}}
	useCase := newUserUseCaseForTest(repo, &testRefreshUseCase{}, storage, inspector, &testBlacklist{}, &testTransaction{})

	if err := useCase.UpdateImage(context.Background(), userID, strings.NewReader("uploaded")); err != nil {
		t.Fatalf("UpdateImage() error = %v", err)
	}
	if !strings.HasPrefix(storage.putKey, "users/"+userID.String()+"/avatars/") || !strings.HasSuffix(storage.putKey, ".png") || string(storage.putContent) != "png-data" || storage.putOptions.MaxBytes != 3*1024*1024 {
		t.Fatalf("UpdateImage() stored wrong key, content, or size limit: key=%q bytes=%q options=%+v", storage.putKey, storage.putContent, storage.putOptions)
	}
	if user.AvatarKey != storage.putKey || len(storage.deletedKeys) != 1 || storage.deletedKeys[0] != "users/old.jpg" {
		t.Fatalf("UpdateImage() did not replace avatar and remove old file: user=%+v deleted=%v", user, storage.deletedKeys)
	}
}

func TestUserAvatarUpdateCleansUploadWhenDatabaseUpdateFails(t *testing.T) {
	userID := uuid.New()
	user := &userdomain.User{ID: userID}
	repo := &rbacUserRepo{users: map[uuid.UUID]*userdomain.User{userID: user}, updateErr: io.ErrClosedPipe}
	storage := &userTestStorage{}
	inspector := &userTestImageInspector{image: port.InspectedImage{Content: strings.NewReader("png-data"), Extension: ".png"}}
	useCase := newUserUseCaseForTest(repo, &testRefreshUseCase{}, storage, inspector, &testBlacklist{}, &testTransaction{})

	if err := useCase.UpdateImage(context.Background(), userID, strings.NewReader("uploaded")); appErrorType(err) != shared.ErrTypeInternal {
		t.Fatalf("UpdateImage() error = %v, want internal error", err)
	}
	if len(storage.deletedKeys) != 1 || storage.deletedKeys[0] != storage.putKey {
		t.Fatalf("UpdateImage() did not clean up new object after database failure: uploaded=%q deleted=%v", storage.putKey, storage.deletedKeys)
	}
}

func TestUserAvatarUpdateMapsInspectionAndStorageErrors(t *testing.T) {
	userID := uuid.New()
	for _, tc := range []struct {
		name       string
		inspectErr error
		putErr     error
		wantErr    shared.ErrorType
	}{
		{name: "invalid image", inspectErr: port.ErrInvalidImage, wantErr: shared.ErrTypeValidation},
		{name: "inspection failure", inspectErr: io.ErrClosedPipe, wantErr: shared.ErrTypeInternal},
		{name: "oversized image", putErr: port.ErrFileTooLarge, wantErr: shared.ErrTypeValidation},
		{name: "storage failure", putErr: io.ErrClosedPipe, wantErr: shared.ErrTypeInternal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &rbacUserRepo{users: map[uuid.UUID]*userdomain.User{userID: {ID: userID}}}
			storage := &userTestStorage{putErr: tc.putErr}
			inspector := &userTestImageInspector{image: port.InspectedImage{Content: strings.NewReader("pixels"), Extension: ".png"}, err: tc.inspectErr}
			useCase := newUserUseCaseForTest(repo, &testRefreshUseCase{}, storage, inspector, &testBlacklist{}, &testTransaction{})
			if err := useCase.UpdateImage(context.Background(), userID, strings.NewReader("upload")); appErrorType(err) != tc.wantErr {
				t.Fatalf("UpdateImage() error=%v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestUserAvatarReadMapsMissingAndInvalidObjects(t *testing.T) {
	userID := uuid.New()
	for _, tc := range []struct {
		name      string
		avatarKey string
		openErr   error
		wantErr   shared.ErrorType
	}{
		{name: "no avatar", wantErr: shared.ErrTypeNotFound},
		{name: "unsupported extension", avatarKey: "users/avatar.gif", wantErr: shared.ErrTypeInternal},
		{name: "missing storage object", avatarKey: "users/avatar.png", openErr: port.ErrFileNotFound, wantErr: shared.ErrTypeNotFound},
		{name: "storage failure", avatarKey: "users/avatar.png", openErr: io.ErrClosedPipe, wantErr: shared.ErrTypeInternal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &rbacUserRepo{users: map[uuid.UUID]*userdomain.User{userID: {ID: userID, AvatarKey: tc.avatarKey}}}
			storage := &userTestStorage{openErr: tc.openErr}
			useCase := newUserUseCaseForTest(repo, &testRefreshUseCase{}, storage, &userTestImageInspector{}, &testBlacklist{}, &testTransaction{})
			if _, err := useCase.GetImage(context.Background(), userID); appErrorType(err) != tc.wantErr {
				t.Fatalf("GetImage() error=%v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestUserAvatarReadAndRemove(t *testing.T) {
	userID := uuid.New()
	user := &userdomain.User{ID: userID, AvatarKey: "users/avatar.jpg"}
	repo := &rbacUserRepo{users: map[uuid.UUID]*userdomain.User{userID: user}}
	storage := &userTestStorage{openResult: port.OpenedFile{Content: io.NopCloser(strings.NewReader("pixels")), Size: 6}}
	useCase := newUserUseCaseForTest(repo, &testRefreshUseCase{}, storage, &userTestImageInspector{}, &testBlacklist{}, &testTransaction{})

	image, err := useCase.GetImage(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetImage() error = %v", err)
	}
	content, err := io.ReadAll(image.Content)
	image.Content.Close()
	if err != nil || string(content) != "pixels" || image.ContentType != "image/jpeg" || image.Size != 6 || storage.openKey != user.AvatarKey {
		t.Fatalf("GetImage() image=%+v content=%q error=%v", image, content, err)
	}
	if err := useCase.RemoveImage(context.Background(), userID); err != nil {
		t.Fatalf("RemoveImage() error = %v", err)
	}
	if user.AvatarKey != "" || len(storage.deletedKeys) != 1 || storage.deletedKeys[0] != "users/avatar.jpg" {
		t.Fatalf("RemoveImage() did not clear avatar and file: user=%+v deleted=%v", user, storage.deletedKeys)
	}
	if err := useCase.RemoveImage(context.Background(), userID); appErrorType(err) != shared.ErrTypeValidation {
		t.Fatalf("RemoveImage() without avatar error = %v, want validation", err)
	}
}

func TestRoleCreateUpdateAndDuplicateErrors(t *testing.T) {
	repo := &rbacRoleRepo{roles: make(map[uuid.UUID]*roledomain.Role)}
	useCase := roleapp.NewRoleUseCase(repo, &testTransaction{}, &testActivityLogRepo{})
	role := &roledomain.Role{Name: "  Support  "}
	if err := useCase.Create(context.Background(), role); err != nil || role.Name != "Support" || repo.queriedName != "Support" || repo.created != role {
		t.Fatalf("Create() role=%+v queried=%q created=%p error=%v", role, repo.queriedName, repo.created, err)
	}

	repo.byName = &roledomain.Role{Name: "Support"}
	if err := useCase.Create(context.Background(), &roledomain.Role{Name: "Support"}); appErrorType(err) != shared.ErrTypeConflict {
		t.Fatalf("Create() duplicate error = %v, want conflict", err)
	}
	repo.byName = nil
	repo.createErr = shared.ErrConflict
	if err := useCase.Create(context.Background(), &roledomain.Role{Name: "Support"}); appErrorType(err) != shared.ErrTypeConflict {
		t.Fatalf("Create() database duplicate error = %v, want conflict", err)
	}

	id := uuid.New()
	repo.roles[id] = &roledomain.Role{ID: id, Name: "Old"}
	updated := &roledomain.Role{ID: id, Name: "  Operator "}
	if err := useCase.Update(context.Background(), updated); err != nil || updated.Name != "Operator" || repo.updated == nil || repo.updated.ID != id || repo.updated.Name != "Operator" {
		t.Fatalf("Update() input=%+v persisted=%+v error=%v", updated, repo.updated, err)
	}
}

func TestRoleDeleteRequiresExistingRoleAndMapsRepositoryErrors(t *testing.T) {
	id := uuid.New()
	t.Run("deletes existing role", func(t *testing.T) {
		repo := &rbacRoleRepo{roles: map[uuid.UUID]*roledomain.Role{id: {ID: id}}}
		if err := roleapp.NewRoleUseCase(repo, &testTransaction{}, &testActivityLogRepo{}).Delete(context.Background(), id); err != nil || repo.deletedID != id {
			t.Fatalf("Delete() deletedID=%v error=%v", repo.deletedID, err)
		}
	})

	t.Run("does not delete a missing role", func(t *testing.T) {
		repo := &rbacRoleRepo{roles: make(map[uuid.UUID]*roledomain.Role)}
		if err := roleapp.NewRoleUseCase(repo, &testTransaction{}, &testActivityLogRepo{}).Delete(context.Background(), id); appErrorType(err) != shared.ErrTypeNotFound || repo.deletedID != uuid.Nil() {
			t.Fatalf("Delete() error=%v deletedID=%v, want not found without repository delete", err, repo.deletedID)
		}
	})

	t.Run("maps repository failure", func(t *testing.T) {
		repo := &rbacRoleRepo{roles: map[uuid.UUID]*roledomain.Role{id: {ID: id}}, deleteErr: io.ErrClosedPipe}
		if err := roleapp.NewRoleUseCase(repo, &testTransaction{}, &testActivityLogRepo{}).Delete(context.Background(), id); appErrorType(err) != shared.ErrTypeInternal || repo.deletedID != id {
			t.Fatalf("Delete() error=%v deletedID=%v, want internal error", err, repo.deletedID)
		}
	})
}

func TestRolePermissionChangesLoadRelationsThroughNamedOperation(t *testing.T) {
	roleID, permissionID := uuid.New(), uuid.New()
	role := &roledomain.Role{ID: roleID, Name: "Editor"}
	repo := &rbacRoleRepo{roles: map[uuid.UUID]*roledomain.Role{roleID: role}}
	useCase := roleapp.NewRoleUseCase(repo, &testTransaction{}, &testActivityLogRepo{})
	permissionIDs := []uuid.UUID{permissionID}

	if err := useCase.AddPermissions(context.Background(), roleID, permissionIDs); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}
	if repo.addedRole.ID != roleID || len(repo.addedPermissionIDs) != 1 || repo.addedPermissionIDs[0] != permissionID {
		t.Fatalf("AddPermissions() passed wrong role or IDs: %+v %v", repo.addedRole, repo.addedPermissionIDs)
	}
	role.Permissions = []permissiondomain.Permission{{ID: permissionID}}
	if err := useCase.RemovePermissions(context.Background(), roleID, permissionIDs); err != nil {
		t.Fatalf("RemovePermissions() error = %v", err)
	}
	if repo.removedRole.ID != roleID || len(repo.removedPermissionIDs) != 1 || repo.removedPermissionIDs[0] != permissionID {
		t.Fatalf("RemovePermissions() passed wrong role or IDs: %+v %v", repo.removedRole, repo.removedPermissionIDs)
	}
	if _, err := useCase.Find(context.Background(), roleID); err != nil || !repo.loadedPermissions {
		t.Fatalf("Find() loaded permissions=%v error=%v", repo.loadedPermissions, err)
	}
}

func TestRoleListValidatesFieldsBeforeRepository(t *testing.T) {
	repo := &rbacRoleRepo{listResult: &shared.PaginatedResult[roledomain.Role]{Page: 1}}
	useCase := roleapp.NewRoleUseCase(repo, &testTransaction{}, &testActivityLogRepo{})
	params := shared.PaginationParams{Page: 1, Limit: 10, Sort: []shared.SortParam{{Field: "name", Direction: shared.SortAsc}}}
	if _, err := useCase.List(context.Background(), params, []shared.Filter{{Field: "name", Operator: shared.OperatorEquals, Value: "Admin"}}); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if repo.listCalls != 1 || repo.listParams.Sort[0].Field != "name" || len(repo.listFilters) != 1 {
		t.Fatalf("List() passed wrong params to repository: calls=%d params=%+v filters=%+v", repo.listCalls, repo.listParams, repo.listFilters)
	}
	if _, err := useCase.List(context.Background(), params, []shared.Filter{{Field: "password", Operator: shared.OperatorEquals}}); appErrorType(err) != shared.ErrTypeValidation || repo.listCalls != 1 {
		t.Fatalf("List() accepted unsafe filter or called repository: error=%v calls=%d", err, repo.listCalls)
	}
}
