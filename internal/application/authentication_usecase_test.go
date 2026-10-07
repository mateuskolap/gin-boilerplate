package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"uuid"

	"gin-boilerplate/internal/domain/shared"
	"gin-boilerplate/internal/infra/security"
	refreshtokendomain "gin-boilerplate/internal/refresh_tokens/domain"
	roledomain "gin-boilerplate/internal/roles/domain"
	userdomain "gin-boilerplate/internal/users/domain"

	"golang.org/x/crypto/bcrypt"
)

func TestRegisterNormalizesHashesAndAssignsDefaultRole(t *testing.T) {
	auth, userRepo, roleRepo, _, _, _ := newTestAuthUseCase()
	roleRepo.byName = &roledomain.Role{Name: roledomain.RoleUser}
	user := &userdomain.User{Name: "  Alice Example ", Email: " Alice@Example.COM ", Password: "valid-password", OrganizationID: uuid.New()}

	if err := auth.Register(context.Background(), user); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if user.Name != "Alice Example" || user.Email != "alice@example.com" {
		t.Fatalf("Register() did not normalize user: %+v", user)
	}
	if user.Password == "valid-password" || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("valid-password")) != nil {
		t.Fatal("Register() did not store a usable password hash")
	}
	if userRepo.lastEmail != "alice@example.com" || len(user.Roles) != 1 || user.Roles[0].Name != roledomain.RoleUser {
		t.Fatalf("Register() used wrong lookup or default role: email=%q roles=%+v", userRepo.lastEmail, user.Roles)
	}
}

func TestRegisterAllowsUserWithoutOrganization(t *testing.T) {
	auth, _, roleRepo, _, _, _ := newTestAuthUseCase()
	roleRepo.byName = &roledomain.Role{Name: roledomain.RoleUser}
	user := &userdomain.User{Name: "Alice Example", Email: "alice@example.com", Password: "valid-password"}

	if err := auth.Register(context.Background(), user); err != nil {
		t.Fatalf("Register() without organization error = %v", err)
	}
	if user.OrganizationID != uuid.Nil() {
		t.Fatalf("Register() organization ID = %s, want no organization", user.OrganizationID)
	}
}

func TestRegisterRejectsInvalidAndDuplicateUsers(t *testing.T) {
	tests := []struct {
		name  string
		user  userdomain.User
		setup func(*testUserRepo)
		want  shared.ErrorType
	}{
		{name: "short name", user: userdomain.User{Name: "A", Email: "a@example.com", Password: "valid-password"}, want: shared.ErrTypeValidation},
		{name: "short password", user: userdomain.User{Name: "Alice", Email: "a@example.com", Password: "short"}, want: shared.ErrTypeValidation},
		{name: "duplicate email", user: userdomain.User{Name: "Alice", Email: "a@example.com", Password: "valid-password"}, setup: func(r *testUserRepo) {
			r.byEmail = &userdomain.User{Email: "a@example.com"}
		}, want: shared.ErrTypeConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, userRepo, _, _, _, _ := newTestAuthUseCase()
			if tt.setup != nil {
				tt.setup(userRepo)
			}
			err := auth.Register(context.Background(), &tt.user)
			if got := appErrorType(err); got != tt.want {
				t.Fatalf("Register() error type = %q, want %q (error %v)", got, tt.want, err)
			}
		})
	}
}

func TestLoginCreatesAccessAndRefreshTokens(t *testing.T) {
	auth, userRepo, _, refresh, _, _ := newTestAuthUseCase()
	userID := uuid.New()
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	userRepo.byEmail = &userdomain.User{ID: userID, Email: "person@example.com", Password: string(hash)}
	refresh.createdToken = "refresh-token"

	tokens, err := auth.Login(context.Background(), " Person@Example.com ", "correct-password", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if tokens.RefreshToken != "refresh-token" || tokens.AccessToken == "" {
		t.Fatalf("Login() returned incomplete tokens: %+v", tokens)
	}
	claims, err := security.ParseAndValidateJWT(tokens.AccessToken, "test-secret-with-at-least-32-characters", "issuer", "audience")
	if err != nil || claims.Subject != userID.String() {
		t.Fatalf("Login() access token claims = %+v, error = %v", claims, err)
	}
	if userRepo.lastEmail != "person@example.com" || refresh.createdUserID != userID || refresh.createdIP != "127.0.0.1" || refresh.createdUserAgent != "test-agent" {
		t.Fatalf("Login() passed wrong user or client metadata: repo=%q refresh=%+v", userRepo.lastEmail, refresh)
	}
}

func TestLoginRejectsMissingUserAndWrongPassword(t *testing.T) {
	for _, withUser := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing user", true: "wrong password"}[withUser], func(t *testing.T) {
			auth, userRepo, _, _, _, _ := newTestAuthUseCase()
			if withUser {
				hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
				if err != nil {
					t.Fatal(err)
				}
				userRepo.byEmail = &userdomain.User{Email: "person@example.com", Password: string(hash)}
			}
			if _, err := auth.Login(context.Background(), "person@example.com", "wrong-password", "", ""); appErrorType(err) != shared.ErrTypeUnauthorized {
				t.Fatalf("Login() error = %v, want unauthorized", err)
			}
		})
	}
}

func TestRefreshRotatesRefreshToken(t *testing.T) {
	auth, userRepo, _, refresh, _, _ := newTestAuthUseCase()
	userID := uuid.New()
	userRepo.byID[userID] = &userdomain.User{ID: userID}
	refresh.validatedToken = &refreshtokendomain.RefreshToken{UserID: userID}
	refresh.rotatedToken = "replacement"

	tokens, err := auth.Refresh(context.Background(), "current", "192.0.2.1", "browser")
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if tokens.RefreshToken != "replacement" || tokens.AccessToken == "" || refresh.validatedInput != "current" || refresh.rotatedOldToken != "current" || refresh.rotatedIP != "192.0.2.1" || refresh.rotatedUserAgent != "browser" {
		t.Fatalf("Refresh() returned or rotated wrong tokens: tokens=%+v refresh=%+v", tokens, refresh)
	}
}

func TestRefreshRejectsMissingUser(t *testing.T) {
	auth, _, _, refresh, _, _ := newTestAuthUseCase()
	refresh.validatedToken = &refreshtokendomain.RefreshToken{UserID: uuid.New()}
	if _, err := auth.Refresh(context.Background(), "current", "", ""); appErrorType(err) != shared.ErrTypeUnauthorized {
		t.Fatalf("Refresh() error = %v, want unauthorized", err)
	}
}

func TestLogoutRevokesAccessAndRefreshTokens(t *testing.T) {
	auth, _, _, refresh, blacklist, _ := newTestAuthUseCase()
	accessToken, err := testAccessToken(uuid.New())
	if err != nil {
		t.Fatal(err)
	}

	if err := auth.Logout(context.Background(), accessToken, "refresh-to-revoke"); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if blacklist.revokedTokenID == "" || blacklist.revokedTokenTTL <= 0 || refresh.revokedToken != "refresh-to-revoke" {
		t.Fatalf("Logout() did not revoke both tokens: blacklist=%+v refresh=%+v", blacklist, refresh)
	}
}

func TestLogoutRejectsInvalidAccessTokenWhenNoRefreshToken(t *testing.T) {
	auth, _, _, _, _, _ := newTestAuthUseCase()
	if err := auth.Logout(context.Background(), "not-a-jwt", ""); appErrorType(err) != shared.ErrTypeUnauthorized {
		t.Fatalf("Logout() error = %v, want unauthorized", err)
	}
}

func TestChangePasswordUpdatesHashAndRevokesAllSessions(t *testing.T) {
	auth, userRepo, _, refresh, blacklist, tx := newTestAuthUseCase()
	userID := uuid.New()
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	userRepo.byID[userID] = &userdomain.User{ID: userID, Password: string(oldHash)}

	if err := auth.ChangePassword(context.Background(), userID, "old-password", "new-password"); err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(userRepo.byID[userID].Password), []byte("new-password")) != nil {
		t.Fatal("ChangePassword() did not store the new password hash")
	}
	if blacklist.revokedUserID != userID.String() || blacklist.revokeUserTTL <= 0 || refresh.revokedAllUserID != userID || tx.calls != 1 {
		t.Fatalf("ChangePassword() did not invalidate all sessions: blacklist=%+v refresh=%+v transactions=%d", blacklist, refresh, tx.calls)
	}
}

func TestChangePasswordRejectsIncorrectOrUnchangedPassword(t *testing.T) {
	auth, userRepo, _, _, _, tx, _ := newTestAuthUseCaseWithPasswordLevel(2)
	userID := uuid.New()
	hash, err := bcrypt.GenerateFromPassword([]byte("Current0!"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	userRepo.byID[userID] = &userdomain.User{ID: userID, Password: string(hash)}

	if err := auth.ChangePassword(context.Background(), userID, "wrong-password", "Nextpass0!"); appErrorType(err) != shared.ErrTypeUnauthorized {
		t.Fatalf("ChangePassword() error = %v, want unauthorized", err)
	}
	if err := auth.ChangePassword(context.Background(), userID, "Current0!", "Current0!"); appErrorType(err) != shared.ErrTypeValidation {
		t.Fatalf("ChangePassword() error = %v, want validation", err)
	}
	if tx.calls != 0 {
		t.Fatalf("ChangePassword() opened %d transactions for invalid requests", tx.calls)
	}

	if err := auth.ChangePassword(context.Background(), userID, "Current0!", "invalid-password!"); appErrorType(err) != shared.ErrTypeValidation {
		t.Fatalf("ChangePassword() error = %v, want validation for level 2", err)
	}
}

func TestValidateAccessTokenChecksBlacklistAndOwner(t *testing.T) {
	auth, userRepo, _, _, blacklist, _ := newTestAuthUseCase()
	userID := uuid.New()
	userRepo.byID[userID] = &userdomain.User{ID: userID}
	token, err := testAccessToken(userID)
	if err != nil {
		t.Fatal(err)
	}
	jwtClaims, err := security.ParseAndValidateJWT(token, "test-secret-with-at-least-32-characters", "issuer", "audience")
	if err != nil {
		t.Fatal(err)
	}

	claims, err := auth.ValidateAccessToken(context.Background(), token)
	if err != nil || claims.Subject != userID.String() || claims.TokenID == "" {
		t.Fatalf("ValidateAccessToken() claims = %+v, error = %v", claims, err)
	}
	if blacklist.checkedTokenID != jwtClaims.ID || blacklist.checkedUserID != jwtClaims.Subject || !blacklist.checkedIssuedAt.Equal(jwtClaims.IssuedAt.Time) {
		t.Fatalf("ValidateAccessToken() checked wrong revocation claims: %+v", blacklist)
	}

	blacklist.tokenIsRevoked = true
	if _, err := auth.ValidateAccessToken(context.Background(), token); appErrorType(err) != shared.ErrTypeUnauthorized {
		t.Fatalf("ValidateAccessToken() error = %v, want unauthorized for revoked token", err)
	}
	blacklist.tokenIsRevoked = false
	blacklist.userIsRevoked = true
	if _, err := auth.ValidateAccessToken(context.Background(), token); appErrorType(err) != shared.ErrTypeUnauthorized {
		t.Fatalf("ValidateAccessToken() error = %v, want unauthorized for revoked user tokens", err)
	}
	blacklist.userIsRevoked = false
	delete(userRepo.byID, userID)
	if _, err := auth.ValidateAccessToken(context.Background(), token); appErrorType(err) != shared.ErrTypeUnauthorized {
		t.Fatalf("ValidateAccessToken() error = %v, want unauthorized for deleted owner", err)
	}
}

func TestRegisterEnforcesPasswordBounds(t *testing.T) {
	tests := []struct {
		name    string
		pw      string
		wantErr bool
	}{
		{name: "too short", pw: "short", wantErr: true},
		{name: "too many bytes", pw: strings.Repeat("x", 73), wantErr: true},
		{name: "minimum length", pw: "12345678"},
		{name: "unicode characters", pw: strings.Repeat("é", 8)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			auth, _, roleRepo, _, _, _ := newTestAuthUseCase()
			roleRepo.byName = &roledomain.Role{Name: roledomain.RoleUser}
			err := auth.Register(context.Background(), &userdomain.User{Name: "Alice", Email: "alice@example.com", Password: tc.pw, OrganizationID: uuid.New()})
			if tc.wantErr && appErrorType(err) != shared.ErrTypeValidation {
				t.Fatalf("Register() error = %v, want validation", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Register() error = %v", err)
			}
		})
	}
}

func TestRegisterAppliesConfiguredPasswordLevel(t *testing.T) {
	tests := []struct {
		name    string
		level   int
		pw      string
		wantErr bool
	}{
		{name: "level one permits eight characters", level: 1, pw: "password"},
		{name: "level two requires character classes", level: 2, pw: "password", wantErr: true},
		{name: "level two permits mixed classes", level: 2, pw: "Passw0rd!"},
		{name: "level two rejects missing uppercase", level: 2, pw: "password1!", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			auth, _, roleRepo, _, _, _, _ := newTestAuthUseCaseWithPasswordLevel(tc.level)
			roleRepo.byName = &roledomain.Role{Name: roledomain.RoleUser}
			err := auth.Register(context.Background(), &userdomain.User{Name: "Alice", Email: "alice@example.com", Password: tc.pw, OrganizationID: uuid.New()})
			if tc.wantErr && appErrorType(err) != shared.ErrTypeValidation {
				t.Fatalf("Register() error = %v, want validation", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Register() error = %v", err)
			}
		})
	}
}

func TestRegisterChecksPwnedPasswordsAtLevelThree(t *testing.T) {
	auth, _, roleRepo, _, _, _, checker := newTestAuthUseCaseWithPasswordLevel(3)
	roleRepo.byName = &roledomain.Role{Name: roledomain.RoleUser}
	checker.compromised = true
	user := &userdomain.User{Name: "Alice", Email: "alice@example.com", Password: "Passw0rd!", OrganizationID: uuid.New()}
	if err := auth.Register(context.Background(), user); appErrorType(err) != shared.ErrTypeValidation {
		t.Fatalf("Register() error = %v, want validation for compromised password", err)
	}

	checker.compromised = false
	checker.err = errors.New("checker unavailable")
	user = &userdomain.User{Name: "Alice", Email: "alice@example.com", Password: "Passw0rd!", OrganizationID: uuid.New()}
	if err := auth.Register(context.Background(), user); err != nil {
		t.Fatalf("Register() error = %v, want fail-open when checker is unavailable", err)
	}
	if checker.calls != 2 {
		t.Fatalf("password checker called %d times, want 2", checker.calls)
	}
}

func TestChangePasswordPropagatesStorageFailure(t *testing.T) {
	auth, userRepo, _, _, _, tx := newTestAuthUseCase()
	userID := uuid.New()
	hash, err := bcrypt.GenerateFromPassword([]byte("current-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	userRepo.byID[userID] = &userdomain.User{ID: userID, Password: string(hash)}
	userRepo.updateErr = context.DeadlineExceeded

	err = auth.ChangePassword(context.Background(), userID, "current-password", "next-password")
	if err == nil || tx.calls != 1 {
		t.Fatalf("ChangePassword() error=%v transaction calls=%d", err, tx.calls)
	}
}
