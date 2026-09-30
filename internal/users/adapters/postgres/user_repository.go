package postgres

import (
	"context"
	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	postgresinfra "gin-boilerplate/internal/infra/postgres"
	rolespostgres "gin-boilerplate/internal/roles/adapters/postgres"
	userdomain "gin-boilerplate/internal/users/domain"
	"uuid"

	"gorm.io/gorm"
)

type userRepository struct {
	*postgresinfra.ActivityLoggingRepository[userdomain.User]
}

func NewUserRepository(db *gorm.DB, activityLogRepo activitylogdomain.ActivityLogRepository) userdomain.UserRepository {
	return &userRepository{
		ActivityLoggingRepository: postgresinfra.NewActivityLoggingRepository(
			db,
			postgresinfra.NewBaseRepository(db,
				func() any { return &UserModel{} },
				func(user *userdomain.User) any { return userModelFromDomain(user) },
				userDomainFromModel,
			),
			activityLogRepo,
		),
	}
}

func (r *userRepository) GetByIDWithRoles(ctx context.Context, id uuid.UUID) (*userdomain.User, error) {
	return r.FindOneBy(ctx, "id = ?", []any{id}, "Roles")
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*userdomain.User, error) {
	return r.FindOneBy(ctx, "email = ?", []any{email})
}

func (r *userRepository) GetByEmailWithRoles(ctx context.Context, email string) (*userdomain.User, error) {
	return r.FindOneBy(ctx, "email = ?", []any{email}, "Roles")
}

func (r *userRepository) AddRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	if len(roleIDs) == 0 {
		return nil
	}

	roles := make([]rolespostgres.RoleModel, len(roleIDs))
	for i, id := range roleIDs {
		roles[i].ID = id
	}

	return r.DB(ctx).WithContext(ctx).
		Model(&UserModel{ID: userID}).
		Association("Roles").Append(&roles)
}

func (r *userRepository) RemoveRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	if len(roleIDs) == 0 {
		return nil
	}

	roles := make([]rolespostgres.RoleModel, len(roleIDs))
	for i, id := range roleIDs {
		roles[i].ID = id
	}

	return r.DB(ctx).WithContext(ctx).
		Model(&UserModel{ID: userID}).
		Association("Roles").Delete(&roles)
}
