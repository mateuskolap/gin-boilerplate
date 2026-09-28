package usecase

import (
	"context"
	"testing"
	"time"
	"uuid"

	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
	"gin-boilerplate/internal/infra/security"
)

func refreshToken(userID uuid.UUID, plaintext string, expiresAt time.Time) *domain.RefreshToken {
	return &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: security.HashSHA256(plaintext),
		ExpiresAt: expiresAt,
	}
}

func newRefreshTokenUseCase(repo *testRefreshRepo, tx *testTransaction) domain.RefreshTokenUseCase {
	return NewRefreshTokenUseCase(repo, tx, time.Hour)
}

func TestRefreshTokenCreateStoresOnlyHashAndExpiry(t *testing.T) {
	repo := &testRefreshRepo{}
	useCase := newRefreshTokenUseCase(repo, &testTransaction{})
	userID := uuid.New()
	startedAt := time.Now().UTC()

	plaintext, err := useCase.Create(context.Background(), userID, "192.0.2.5", "test-agent")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if plaintext == "" || len(repo.tokens) != 1 {
		t.Fatalf("Create() plaintext=%q persisted tokens=%d", plaintext, len(repo.tokens))
	}
	stored := repo.tokens[0]
	if stored.TokenHash != security.HashSHA256(plaintext) || stored.TokenHash == plaintext {
		t.Fatalf("refresh token was not persisted as a hash: %+v", stored)
	}
	if stored.UserID != userID || stored.IpAddress != "192.0.2.5" || stored.UserAgent != "test-agent" || stored.ID == uuid.Nil() {
		t.Fatalf("Create() stored wrong token metadata: %+v", stored)
	}
	if stored.ExpiresAt.Before(startedAt.Add(time.Hour)) || stored.ExpiresAt.After(time.Now().UTC().Add(time.Hour)) {
		t.Fatalf("Create() expiry = %v, outside expected range", stored.ExpiresAt)
	}
}

func TestRefreshTokenValidate(t *testing.T) {
	userID := uuid.New()
	tests := []struct {
		name           string
		plaintext      string
		token          *domain.RefreshToken
		wantErr        shared.ErrorType
		wantRevocation bool
	}{
		{name: "empty token", wantErr: shared.ErrTypeValidation},
		{name: "unknown token", plaintext: "unknown", wantErr: shared.ErrTypeUnauthorized},
		{name: "expired token", plaintext: "expired", token: refreshToken(userID, "expired", time.Now().UTC().Add(-time.Minute)), wantErr: shared.ErrTypeUnauthorized},
		{name: "revoked token", plaintext: "revoked", token: revokedRefreshToken(userID, "revoked"), wantErr: shared.ErrTypeUnauthorized, wantRevocation: true},
		{name: "active token", plaintext: "active", token: refreshToken(userID, "active", time.Now().UTC().Add(time.Hour))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &testRefreshRepo{}
			if tt.token != nil {
				repo.tokens = append(repo.tokens, tt.token)
			}
			useCase := newRefreshTokenUseCase(repo, &testTransaction{})
			got, err := useCase.Validate(context.Background(), tt.plaintext)
			if tt.wantErr != "" {
				if appErrorType(err) != tt.wantErr {
					t.Fatalf("Validate() error = %v, want type %q", err, tt.wantErr)
				}
			} else if err != nil || got != tt.token {
				t.Fatalf("Validate() token=%p, error=%v; want %p", got, err, tt.token)
			}
			if tt.wantRevocation && (len(repo.revokedUserIDs) != 1 || repo.revokedUserIDs[0] != userID) {
				t.Fatalf("Validate() revoked user IDs = %v, want %v", repo.revokedUserIDs, userID)
			}
		})
	}
}

func revokedRefreshToken(userID uuid.UUID, plaintext string) *domain.RefreshToken {
	token := refreshToken(userID, plaintext, time.Now().UTC().Add(time.Hour))
	now := time.Now().UTC()
	token.RevokedAt = &now
	return token
}

func TestRefreshTokenRotateReplacesAndLinksToken(t *testing.T) {
	userID := uuid.New()
	expiresAt := time.Now().UTC().Add(time.Hour)
	old := refreshToken(userID, "old-token", expiresAt)
	repo := &testRefreshRepo{tokens: []*domain.RefreshToken{old}}
	tx := &testTransaction{}
	useCase := newRefreshTokenUseCase(repo, tx)

	replacement, err := useCase.Rotate(context.Background(), "old-token", "198.51.100.2", "new-agent")
	if err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}
	if replacement == "" || replacement == "old-token" || tx.calls != 1 || len(repo.tokens) != 2 {
		t.Fatalf("Rotate() replacement=%q transactions=%d stored tokens=%d", replacement, tx.calls, len(repo.tokens))
	}
	newToken, err := repo.FindByTokenHash(context.Background(), security.HashSHA256(replacement))
	if err != nil || newToken == nil {
		t.Fatalf("replacement token not stored by hash: token=%+v error=%v", newToken, err)
	}
	if old.RevokedAt == nil || old.ReplacedBy == nil || *old.ReplacedBy != newToken.ID {
		t.Fatalf("old token was not revoked and linked to its replacement: %+v", old)
	}
	if !newToken.ExpiresAt.Equal(expiresAt) || newToken.IpAddress != "198.51.100.2" || newToken.UserAgent != "new-agent" {
		t.Fatalf("replacement token lost expiry or request metadata: %+v", newToken)
	}
}

func TestRefreshTokenRotateRejectsReplayAndRevokesUserSessions(t *testing.T) {
	userID := uuid.New()
	repo := &testRefreshRepo{tokens: []*domain.RefreshToken{revokedRefreshToken(userID, "replayed")}}
	useCase := newRefreshTokenUseCase(repo, &testTransaction{})

	if _, err := useCase.Rotate(context.Background(), "replayed", "", ""); appErrorType(err) != shared.ErrTypeUnauthorized {
		t.Fatalf("Rotate() error = %v, want unauthorized", err)
	}
	if len(repo.revokedUserIDs) != 1 || repo.revokedUserIDs[0] != userID {
		t.Fatalf("replay did not revoke all sessions for the token owner: %v", repo.revokedUserIDs)
	}
}

func TestRevokeSessionAndOtherSessions(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	session := refreshToken(userID, "current-session", time.Now().UTC().Add(time.Hour))
	repo := &testRefreshRepo{tokens: []*domain.RefreshToken{session}}
	useCase := newRefreshTokenUseCase(repo, &testTransaction{})

	if err := useCase.RevokeSession(context.Background(), userID, session.ID); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	if session.RevokedAt == nil {
		t.Fatal("RevokeSession() left the session active")
	}
	if err := useCase.RevokeSession(context.Background(), otherUserID, session.ID); appErrorType(err) != shared.ErrTypeNotFound {
		t.Fatalf("RevokeSession() error = %v, want not found for another user", err)
	}

	current := refreshToken(userID, "keep-me", time.Now().UTC().Add(time.Hour))
	repo.tokens = append(repo.tokens, current)
	if err := useCase.RevokeOtherSessions(context.Background(), userID, "keep-me"); err != nil {
		t.Fatalf("RevokeOtherSessions() error = %v", err)
	}
	if repo.revokeExceptUserID != userID || repo.revokeExceptID != current.ID {
		t.Fatalf("RevokeOtherSessions() excluded wrong session: user=%v id=%v", repo.revokeExceptUserID, repo.revokeExceptID)
	}
	if err := useCase.RevokeOtherSessions(context.Background(), otherUserID, "keep-me"); appErrorType(err) != shared.ErrTypeUnauthorized {
		t.Fatalf("RevokeOtherSessions() error = %v, want unauthorized for another owner", err)
	}
	for _, token := range []string{"", "missing", "expired", "revoked"} {
		if token == "expired" {
			repo.tokens = append(repo.tokens, refreshToken(userID, token, time.Now().UTC().Add(-time.Minute)))
		}
		if token == "revoked" {
			repo.tokens = append(repo.tokens, revokedRefreshToken(userID, token))
		}
		if err := useCase.RevokeOtherSessions(context.Background(), userID, token); appErrorType(err) != shared.ErrTypeUnauthorized {
			t.Errorf("RevokeOtherSessions(%q) error = %v, want unauthorized", token, err)
		}
	}
}

func TestRefreshTokenRevokeAndRevokeAllByUserID(t *testing.T) {
	userID := uuid.New()
	token := refreshToken(userID, "revoke-me", time.Now().UTC().Add(time.Hour))
	repo := &testRefreshRepo{tokens: []*domain.RefreshToken{token}}
	useCase := newRefreshTokenUseCase(repo, &testTransaction{})

	if err := useCase.Revoke(context.Background(), "revoke-me"); err != nil || token.RevokedAt == nil {
		t.Fatalf("Revoke() revokedAt=%v error=%v", token.RevokedAt, err)
	}
	if err := useCase.Revoke(context.Background(), "unknown"); appErrorType(err) != shared.ErrTypeNotFound {
		t.Fatalf("Revoke() unknown token error=%v, want not found", err)
	}

	if err := useCase.RevokeAllByUserID(context.Background(), userID); err != nil || len(repo.revokedUserIDs) != 1 || repo.revokedUserIDs[0] != userID {
		t.Fatalf("RevokeAllByUserID() revoked=%v error=%v", repo.revokedUserIDs, err)
	}

	repo.revokeErr = context.DeadlineExceeded
	active := refreshToken(userID, "repo-error", time.Now().UTC().Add(time.Hour))
	repo.tokens = append(repo.tokens, active)
	if err := useCase.Revoke(context.Background(), "repo-error"); appErrorType(err) != shared.ErrTypeInternal {
		t.Fatalf("Revoke() repository error=%v, want internal", err)
	}
	repo.revokeAllErr = context.DeadlineExceeded
	if err := useCase.RevokeAllByUserID(context.Background(), userID); appErrorType(err) != shared.ErrTypeInternal {
		t.Fatalf("RevokeAllByUserID() repository error=%v, want internal", err)
	}
}

func TestListActiveSessionsAddsActiveFiltersAndRejectsUnsafeFilters(t *testing.T) {
	userID := uuid.New()
	repo := &testRefreshRepo{listResult: &shared.PaginatedResult[domain.RefreshToken]{Page: 1, Limit: 10}}
	useCase := newRefreshTokenUseCase(repo, &testTransaction{})
	params := shared.PaginationParams{Page: 1, Limit: 10, Sort: []shared.SortParam{{Field: "created_at", Direction: shared.SortDesc}}}
	filters := []shared.Filter{
		{Field: "user_agent", Operator: shared.OperatorILike, Value: "%browser%"},
		{Field: "expires_at", Operator: shared.OperatorGreaterThan, Value: time.Now()},
	}

	if _, err := useCase.ListActiveByUserID(context.Background(), userID, params, filters); err != nil {
		t.Fatalf("ListActiveByUserID() error = %v", err)
	}
	if repo.listCalls != 1 || repo.listUserID != userID || repo.listParams.Sort[0].Field != "created_at" {
		t.Fatalf("ListActiveByUserID() called repository with wrong user or params: user=%v params=%+v", repo.listUserID, repo.listParams)
	}
	gotFilters := make(map[string]shared.Filter, len(repo.listFilters))
	for _, filter := range repo.listFilters {
		gotFilters[filter.Field] = filter
	}
	if len(gotFilters) != 3 || gotFilters["user_agent"].Operator != shared.OperatorILike || gotFilters["user_agent"].Value != "%browser%" || gotFilters["expires_at"].Operator != shared.OperatorGreaterThan || gotFilters["revoked_at"].Operator != shared.OperatorIsNull {
		t.Fatalf("ListActiveByUserID() filters = %+v, want requested user agent and only active sessions", repo.listFilters)
	}
	if expiresAt, ok := gotFilters["expires_at"].Value.(time.Time); !ok || expiresAt.IsZero() {
		t.Fatalf("ListActiveByUserID() expires_at filter = %#v, want current time", gotFilters["expires_at"].Value)
	}
	if _, err := useCase.ListActiveByUserID(context.Background(), userID, params, []shared.Filter{{Field: "token_hash", Operator: shared.OperatorEquals}}); appErrorType(err) != shared.ErrTypeValidation {
		t.Fatalf("ListActiveByUserID() error = %v, want validation for unsafe filter", err)
	}
	if repo.listCalls != 1 {
		t.Fatalf("repository called %d times after rejecting unsafe filter", repo.listCalls)
	}
}
