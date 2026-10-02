package http

import (
	"fmt"
	_ "gin-boilerplate/docs"
	activityloghttp "gin-boilerplate/internal/activity_logs/adapters/http"
	"gin-boilerplate/internal/delivery/http/middleware"
	v1 "gin-boilerplate/internal/delivery/http/v1"
	"gin-boilerplate/internal/domain/port"
	organizationhttp "gin-boilerplate/internal/organizations/adapters/http"
	permissionhttp "gin-boilerplate/internal/permissions/adapters/http"
	permissiondomain "gin-boilerplate/internal/permissions/domain"
	refreshhttp "gin-boilerplate/internal/refresh_tokens/adapters/http"
	rolehttp "gin-boilerplate/internal/roles/adapters/http"
	userhttp "gin-boilerplate/internal/users/adapters/http"
	userdomain "gin-boilerplate/internal/users/domain"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type RouterConfig struct {
	AuthHandler         *userhttp.AuthHandler
	UserHandler         *userhttp.UserHandler
	RoleHandler         *rolehttp.RoleHandler
	PermissionHandler   *permissionhttp.PermissionHandler
	ActivityLogHandler  *activityloghttp.ActivityLogHandler
	RefreshTokenHandler *refreshhttp.RefreshTokenHandler
	OrganizationHandler *organizationhttp.OrganizationHandler
	HealthHandler       *v1.HealthHandler
	AuthUseCase         userdomain.AuthUseCase
	PermissionChecker   permissiondomain.PermissionCheckerUseCase
	RateLimiter         port.RateLimiter
	TrustedProxies      []string
	CORSAllowedOrigins  []string
	UseAuthCookies      bool
	Env                 string
}

func SetupRouter(cfg RouterConfig) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}

	r.Use(
		middleware.RequestID(),
		middleware.RequestIP(),
		middleware.RequestLogger(),
		middleware.Recovery(),
		middleware.ErrorHandler(),
		middleware.SecurityHeaders(),
	)

	if len(cfg.CORSAllowedOrigins) > 0 {
		r.Use(cors.New(cors.Config{
			AllowOrigins:     cfg.CORSAllowedOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
			ExposeHeaders:    []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
			AllowCredentials: cfg.UseAuthCookies,
			MaxAge:           12 * time.Hour,
		}))
	}

	r.GET("/health/live", cfg.HealthHandler.Liveness)
	r.GET("/health/ready", cfg.HealthHandler.Readiness)

	if cfg.Env != "production" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	requirePermission := func(permission permissiondomain.PermissionName) gin.HandlerFunc {
		return middleware.RequirePermission(permission, cfg.PermissionChecker)
	}
	rateLimit := func(limit int, window time.Duration) gin.HandlerFunc {
		return middleware.RateLimiter(cfg.RateLimiter, limit, window)
	}
	rateLimitByUser := func(limit int, window time.Duration) gin.HandlerFunc {
		return middleware.RateLimiterByUser(cfg.RateLimiter, limit, window)
	}

	api := r.Group("/api/v1")
	if cfg.UseAuthCookies {
		api.Use(middleware.ValidateOrigin(cfg.CORSAllowedOrigins))
	}
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", rateLimit(60, time.Minute), cfg.AuthHandler.Register)
			auth.POST("/login", rateLimit(60, time.Minute), cfg.AuthHandler.Login)
			auth.POST("/refresh", rateLimit(60, time.Minute), cfg.AuthHandler.Refresh)
		}

		protected := api.Group("")
		protected.Use(middleware.AuthenticationMiddleware(cfg.AuthUseCase, cfg.UseAuthCookies))
		{
			protected.POST("/auth/logout", cfg.AuthHandler.Logout)
			protected.GET("/auth/sessions", rateLimitByUser(10, time.Minute), cfg.RefreshTokenHandler.ListRefreshTokensByAuthUser)
			protected.DELETE("/auth/sessions/:id", rateLimitByUser(10, time.Minute), cfg.RefreshTokenHandler.RevokeSession)
			protected.DELETE("/auth/sessions/revoke-others", rateLimitByUser(10, time.Minute), cfg.RefreshTokenHandler.RevokeOtherSessions)
			protected.PATCH("/auth/password", rateLimitByUser(5, time.Minute), cfg.AuthHandler.ChangePassword)

			users := protected.Group("/users")
			{
				users.GET("/profile", cfg.UserHandler.GetProfile)
				users.PUT("/profile", cfg.UserHandler.UpdateProfile)
				users.PUT("/profile/image", cfg.UserHandler.UpdateImage)
				users.DELETE("/profile/image", cfg.UserHandler.RemoveImage)
				users.GET("/:id/image", middleware.RequirePermissionOrOwner(
					permissiondomain.PermissionViewUser,
					cfg.PermissionChecker,
					middleware.OwnerFromUserIDParam("id"),
				), cfg.UserHandler.GetImage)
				users.PUT("/:id", requirePermission(permissiondomain.PermissionUpdateUser), cfg.UserHandler.UpdateUser)
				users.GET("", requirePermission(permissiondomain.PermissionViewUser), cfg.UserHandler.ListUsers)
				users.GET("/:id", requirePermission(permissiondomain.PermissionViewUser), cfg.UserHandler.FindUser)
				users.DELETE("/:id", requirePermission(permissiondomain.PermissionDeleteUser), cfg.UserHandler.DeleteUser)
				users.POST("/:id/roles", requirePermission(permissiondomain.PermissionAddUserRole), cfg.UserHandler.AddRoles)
				users.DELETE("/:id/roles", requirePermission(permissiondomain.PermissionRemoveUserRole), cfg.UserHandler.RemoveRoles)
			}

			roles := protected.Group("/roles")
			{
				roles.GET("", requirePermission(permissiondomain.PermissionViewRole), cfg.RoleHandler.ListRoles)
				roles.POST("", requirePermission(permissiondomain.PermissionCreateRole), cfg.RoleHandler.CreateRole)
				roles.GET("/:id", requirePermission(permissiondomain.PermissionViewRole), cfg.RoleHandler.FindRole)
				roles.PUT("/:id", requirePermission(permissiondomain.PermissionUpdateRole), cfg.RoleHandler.UpdateRole)
				roles.DELETE("/:id", requirePermission(permissiondomain.PermissionDeleteRole), cfg.RoleHandler.DeleteRole)
				roles.POST("/:id/permissions", requirePermission(permissiondomain.PermissionAddRolePermission), cfg.RoleHandler.AddPermissions)
				roles.DELETE("/:id/permissions", requirePermission(permissiondomain.PermissionRemoveRolePermission), cfg.RoleHandler.RemovePermissions)
			}

			permissions := protected.Group("/permissions")
			{
				permissions.GET("", requirePermission(permissiondomain.PermissionViewPermission), cfg.PermissionHandler.ListPermissions)
			}

			organizations := protected.Group("/organizations")
			{
				organizations.GET("", requirePermission(permissiondomain.PermissionViewOrganization), cfg.OrganizationHandler.ListOrganizations)
				organizations.POST("", requirePermission(permissiondomain.PermissionCreateOrganization), cfg.OrganizationHandler.CreateOrganization)
				organizations.GET("/:id", requirePermission(permissiondomain.PermissionViewOrganization), cfg.OrganizationHandler.FindOrganization)
				organizations.PUT("/:id", requirePermission(permissiondomain.PermissionUpdateOrganization), cfg.OrganizationHandler.UpdateOrganization)
				organizations.DELETE("/:id", requirePermission(permissiondomain.PermissionDeleteOrganization), cfg.OrganizationHandler.DeleteOrganization)
			}

			protected.GET("/activity-logs", requirePermission(permissiondomain.PermissionViewActivityLog), cfg.ActivityLogHandler.ListActivityLogs)
		}
	}

	return r, nil
}
