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
	AuthHandler         *v1.AuthHandler
	UserHandler         *v1.UserHandler
	RoleHandler         *v1.RoleHandler
	PermissionHandler   *v1.PermissionHandler
	RefreshTokenHandler *v1.RefreshTokenHandler
	TokenBlacklist      domain.TokenBlackListRepository
	PermissionChecker   domain.PermissionCheckerUseCase
	JWTSecret           string
	Env                 string
}

func SetupRouter(cfg RouterConfig) *gin.Engine {
	r := gin.Default()

	r.Use(middleware.ErrorHandler())

	if cfg.Env != "production" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	requirePermission := func(perm domain.PermissionName) gin.HandlerFunc {
		return middleware.RequirePermission(perm, cfg.PermissionChecker)
	}

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
			protected.GET("/auth/sessions", cfg.RefreshTokenHandler.ListRefreshTokensByAuthUser)

			users := protected.Group("/users")
			{
				users.GET("/profile", cfg.UserHandler.GetProfile)
				users.PUT("/profile", cfg.UserHandler.UpdateProfile)
				users.GET("", requirePermission(domain.PermissionViewUser), cfg.UserHandler.ListUsers)
				users.GET("/:id", requirePermission(domain.PermissionViewUser), cfg.UserHandler.FindUser)
				users.POST("/:id/roles", requirePermission(domain.PermissionAddUserRole), cfg.UserHandler.AddRoles)
				users.DELETE("/:id/roles", requirePermission(domain.PermissionRemoveUserRole), cfg.UserHandler.RemoveRoles)
			}

			roles := protected.Group("/roles")
			{
				roles.GET("", requirePermission(domain.PermissionViewRole), cfg.RoleHandler.ListRoles)
				roles.POST("", requirePermission(domain.PermissionCreateRole), cfg.RoleHandler.CreateRole)
				roles.GET("/:id", requirePermission(domain.PermissionViewRole), cfg.RoleHandler.FindRole)
				roles.PUT("/:id", requirePermission(domain.PermissionUpdateRole), cfg.RoleHandler.UpdateRole)
				roles.DELETE("/:id", requirePermission(domain.PermissionDeleteRole), cfg.RoleHandler.DeleteRole)
				roles.POST("/:id/permissions", requirePermission(domain.PermissionAddRolePermission), cfg.RoleHandler.AddPermissions)
				roles.DELETE("/:id/permissions", requirePermission(domain.PermissionRemoveRolePermission), cfg.RoleHandler.RemovePermissions)
			}

			permissions := protected.Group("/permissions")
			{
				permissions.GET("", requirePermission(domain.PermissionViewPermission), cfg.PermissionHandler.ListPermissions)
			}
		}
	}

	return r
}
