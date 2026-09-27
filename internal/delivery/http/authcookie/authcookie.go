package authcookie

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	AccessTokenName  = "access_token"
	RefreshTokenName = "refresh_token"
	accessTokenPath  = "/api/v1"
	refreshTokenPath = "/api/v1/auth"
)

func Set(c *gin.Context, accessToken, refreshToken string, accessTTL, refreshTTL time.Duration, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(AccessTokenName, accessToken, int(accessTTL.Seconds()), accessTokenPath, "", secure, true)
	c.SetCookie(RefreshTokenName, refreshToken, int(refreshTTL.Seconds()), refreshTokenPath, "", secure, true)
}

func Clear(c *gin.Context, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(AccessTokenName, "", -1, accessTokenPath, "", secure, true)
	c.SetCookie(RefreshTokenName, "", -1, refreshTokenPath, "", secure, true)
}

func Get(c *gin.Context, name string) string {
	value, _ := c.Cookie(name)
	return value
}
