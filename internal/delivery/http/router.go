package http

import (
	_ "gin-boilerplate/docs"
	"gin-boilerplate/internal/delivery/http/middleware"
	v1 "gin-boilerplate/internal/delivery/http/v1"
	"gin-boilerplate/internal/domain"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type RouterConfig struct {
	AuthHandler       *v1.AuthHandler
	UserHandler       *v1.UserHandler
	RoleHandler       *v1.RoleHandler
	PermissionHandler *v1.PermissionHandler
	TokenBlacklist    domain.TokenBlackList
	JWTSecret         string
	Env               string
}

// HealthCheck godoc
// @Summary      Health check
// @Description  Check if the API server is alive and running
// @Tags         Health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

func SetupRouter(cfg RouterConfig) *gin.Engine {
	r := gin.Default()

	r.Use(middleware.ErrorHandler())

	if cfg.Env != "production" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	r.GET("/health", healthCheck)

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", cfg.AuthHandler.Register)
			auth.POST("/login", cfg.AuthHandler.Login)
			auth.POST("/refresh", cfg.AuthHandler.Refresh)
		}

		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(cfg.JWTSecret, cfg.TokenBlacklist))
		{
			protected.POST("/auth/logout", cfg.AuthHandler.Logout)

			users := protected.Group("/users")
			{
				users.GET("/profile", cfg.UserHandler.GetProfile)
				users.PUT("/profile", cfg.UserHandler.UpdateProfile)
				users.GET("", cfg.UserHandler.ListUsers)
				users.GET("/:id", cfg.UserHandler.FindUser)
				users.POST("/:id/roles", cfg.UserHandler.AddRoles)
				users.DELETE("/:id/roles", cfg.UserHandler.RemoveRoles)
			}

			roles := protected.Group("/roles")
			{
				roles.GET("", cfg.RoleHandler.ListRoles)
				roles.POST("", cfg.RoleHandler.CreateRole)
				roles.GET("/:id", cfg.RoleHandler.FindRole)
				roles.PUT("/:id", cfg.RoleHandler.UpdateRole)
				roles.DELETE("/:id", cfg.RoleHandler.DeleteRole)
				roles.POST("/:id/permissions", cfg.RoleHandler.AddPermissions)
				roles.DELETE("/:id/permissions", cfg.RoleHandler.RemovePermissions)
			}

			permissions := protected.Group("/permissions")
			{
				permissions.GET("", cfg.PermissionHandler.ListPermissions)
			}
		}
	}

	return r
}
