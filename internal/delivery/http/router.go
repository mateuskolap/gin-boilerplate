package http

import (
	"gin-boilerplate/internal/delivery/http/middleware"
	v1 "gin-boilerplate/internal/delivery/http/v1"
	"gin-boilerplate/internal/domain"

	"github.com/gin-gonic/gin"
)

type RouterConfig struct {
	AuthHandler    *v1.AuthHandler
	UserHandler    *v1.UserHandler
	TokenBlacklist domain.TokenBlackList
	JWTSecret      string
}

func SetupRouter(cfg RouterConfig) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", cfg.AuthHandler.Register)
			auth.POST("/login", cfg.AuthHandler.Login)
		}

		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(cfg.JWTSecret, cfg.TokenBlacklist))
		{
			protected.POST("/auth/logout", cfg.AuthHandler.Logout)

			users := protected.Group("/users")
			{
				users.GET("/profile", cfg.UserHandler.GetProfile)
				users.PUT("/profile", cfg.UserHandler.UpdateProfile)
			}
		}
	}

	return r
}
