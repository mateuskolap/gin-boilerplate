package security

import (
	"encoding/hex"
	"testing"
	"time"
	"uuid"
)

func TestRandomTokenAndHash(t *testing.T) {
	first, err := GenerateRandomToken(32)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := hex.DecodeString(first)
	if err != nil || len(decoded) != 32 {
		t.Fatalf("GenerateRandomToken() = %q, decoded bytes = %d, error = %v", first, len(decoded), err)
	}
	second, err := GenerateRandomToken(32)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("GenerateRandomToken() returned the same token twice")
	}
	if got, want := HashSHA256("abc"), "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"; got != want {
		t.Fatalf("HashSHA256() = %s, want %s", got, want)
	}
}

func TestAccessTokenValidation(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret-with-at-least-32-characters"
	token, err := GenerateAccessToken(userID, secret, "issuer", "audience", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseAndValidateJWT(token, secret, "issuer", "audience")
	if err != nil {
		t.Fatalf("ParseAndValidateJWT() error = %v", err)
	}
	if claims.Subject != userID.String() || claims.ID == "" || claims.IssuedAt == nil || claims.ExpiresAt == nil {
		t.Fatalf("unexpected claims: %+v", claims.RegisteredClaims)
	}

	tests := []struct {
		name                     string
		secret, issuer, audience string
		token                    string
	}{
		{name: "wrong secret", secret: "different-secret", issuer: "issuer", audience: "audience", token: token},
		{name: "wrong issuer", secret: secret, issuer: "other", audience: "audience", token: token},
		{name: "wrong audience", secret: secret, issuer: "issuer", audience: "other", token: token},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseAndValidateJWT(tt.token, tt.secret, tt.issuer, tt.audience); err == nil {
				t.Fatal("ParseAndValidateJWT() accepted invalid token")
			}
		})
	}

	expired, err := GenerateAccessToken(userID, secret, "issuer", "audience", -time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseAndValidateJWT(expired, secret, "issuer", "audience"); err == nil {
		t.Fatal("ParseAndValidateJWT() accepted expired token")
	}
}
