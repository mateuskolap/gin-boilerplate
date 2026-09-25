package http

import (
	"fmt"
	_ "gin-boilerplate/docs"
	"gin-boilerplate/internal/delivery/http/middleware"
	v1 "gin-boilerplate/internal/delivery/http/v1"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/port"
	"time"

	"github.com/gin-contrib/cors"
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
	HealthHandler       *v1.HealthHandler
	AuthUseCase         domain.AuthUseCase
	PermissionChecker   domain.PermissionCheckerUseCase
	RateLimiter         port.RateLimiter
	TrustedProxies      []string
	CORSAllowedOrigins  []string
	Env                 string
}

func SetupRouter(cfg RouterConfig) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}

	r.Use(
		middleware.RequestID(),
		middleware.RequestLogger(),
		gin.Recovery(),
		middleware.ErrorHandler(),
		middleware.SecurityHeaders(),
	)

	if len(cfg.CORSAllowedOrigins) > 0 {
		r.Use(cors.New(cors.Config{
			AllowOrigins:     cfg.CORSAllowedOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
			ExposeHeaders:    []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
			AllowCredentials: false,
			MaxAge:           12 * time.Hour,
		}))
	}

	r.GET("/health/live", cfg.HealthHandler.Liveness)
	r.GET("/health/ready", cfg.HealthHandler.Readiness)

	if cfg.Env != "production" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	requirePermission := func(permission domain.PermissionName) gin.HandlerFunc {
		return middleware.RequirePermission(permission, cfg.PermissionChecker)
	}
	rateLimit := func(limit int, window time.Duration) gin.HandlerFunc {
		return middleware.RateLimiter(cfg.RateLimiter, limit, window)
	}

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", rateLimit(5, time.Minute), cfg.AuthHandler.Register)
			auth.POST("/login", rateLimit(10, time.Minute), cfg.AuthHandler.Login)
			auth.POST("/refresh", rateLimit(30, time.Minute), cfg.AuthHandler.Refresh)
		}

		protected := api.Group("")
		protected.Use(middleware.AuthenticationMiddleware(cfg.AuthUseCase))
		{
			protected.POST("/auth/logout", cfg.AuthHandler.Logout)
			protected.GET("/auth/sessions", cfg.RefreshTokenHandler.ListRefreshTokensByAuthUser)

			users := protected.Group("/users")
			{
				users.GET("/profile", cfg.UserHandler.GetProfile)
				users.PUT("/profile", cfg.UserHandler.UpdateProfile)
				users.PUT("/profile/image", cfg.UserHandler.UpdateImage)
				users.DELETE("/profile/image", cfg.UserHandler.RemoveImage)
				users.GET("/:id/image", middleware.RequirePermissionOrOwner(
					domain.PermissionViewUser,
					cfg.PermissionChecker,
					middleware.OwnerFromUserIDParam("id"),
				), cfg.UserHandler.GetImage)
				users.PUT("/:id", requirePermission(domain.PermissionUpdateUser), cfg.UserHandler.UpdateUser)
				users.GET("", requirePermission(domain.PermissionViewUser), cfg.UserHandler.ListUsers)
				users.GET("/:id", requirePermission(domain.PermissionViewUser), cfg.UserHandler.FindUser)
				users.DELETE("/:id", requirePermission(domain.PermissionDeleteUser), cfg.UserHandler.DeleteUser)
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

	return r, nil
}
