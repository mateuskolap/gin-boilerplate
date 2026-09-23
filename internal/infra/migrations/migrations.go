package migrations

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

const initialMigrationID = "20260923000000"

func Migrate(db *gorm.DB) error {
	return newMigrator(db).Migrate()
}

func RollbackLast(db *gorm.DB) error {
	return newMigrator(db).RollbackLast()
}

func newMigrator(db *gorm.DB) *gormigrate.Gormigrate {
	options := *gormigrate.DefaultOptions
	options.TableName = "schema_migrations"
	options.UseTransaction = true
	options.ValidateUnknownMigrations = true
	return gormigrate.New(db, &options, []*gormigrate.Migration{{
		ID: initialMigrationID,
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(
				&initialUser{},
				&initialRole{},
				&initialPermission{},
				&initialUserRole{},
				&initialRolePermission{},
				&initialRefreshToken{},
			)
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(
				"refresh_tokens",
				"role_permissions",
				"user_roles",
				"permissions",
				"roles",
				"users",
			)
		},
	}})
}

type InitialModel struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP"`
}

type initialUser struct {
	InitialModel
	DeletedAt *time.Time `gorm:"type:timestamptz;index:idx_users_deleted_at"`
	Name      string     `gorm:"type:varchar(100);not null"`
	Email     string     `gorm:"type:varchar(320);not null;uniqueIndex:idx_users_email_active,expression:LOWER(email),where:deleted_at IS NULL"`
	Password  string     `gorm:"type:varchar(255);not null"`
}

func (initialUser) TableName() string { return "users" }

type initialRole struct {
	InitialModel
	Name string `gorm:"type:varchar(100);not null;uniqueIndex:roles_name_key"`
}

func (initialRole) TableName() string { return "roles" }

type initialPermission struct {
	InitialModel
	Name string `gorm:"type:varchar(100);not null;uniqueIndex:permissions_name_key"`
}

func (initialPermission) TableName() string { return "permissions" }

type initialUserRole struct {
	UserID string      `gorm:"type:uuid;not null;primaryKey"`
	RoleID string      `gorm:"type:uuid;not null;primaryKey;index:idx_user_roles_role_id"`
	User   initialUser `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Role   initialRole `gorm:"foreignKey:RoleID;references:ID;constraint:OnDelete:CASCADE"`
}

func (initialUserRole) TableName() string { return "user_roles" }

type initialRolePermission struct {
	RoleID       string            `gorm:"type:uuid;not null;primaryKey"`
	PermissionID string            `gorm:"type:uuid;not null;primaryKey;index:idx_role_permissions_permission_id"`
	Role         initialRole       `gorm:"foreignKey:RoleID;references:ID;constraint:OnDelete:CASCADE"`
	Permission   initialPermission `gorm:"foreignKey:PermissionID;references:ID;constraint:OnDelete:CASCADE"`
}

func (initialRolePermission) TableName() string { return "role_permissions" }

type initialRefreshToken struct {
	InitialModel
	UserID     string      `gorm:"type:uuid;not null;index:idx_refresh_tokens_user_id;index:idx_refresh_tokens_active_user,priority:1,where:revoked_at IS NULL"`
	TokenHash  string      `gorm:"type:varchar(64);not null;uniqueIndex:refresh_tokens_token_hash_key"`
	ExpiresAt  time.Time   `gorm:"type:timestamptz;not null;index:idx_refresh_tokens_expires_at;index:idx_refresh_tokens_active_user,priority:2,where:revoked_at IS NULL"`
	RevokedAt  *time.Time  `gorm:"type:timestamptz;default:null"`
	ReplacedBy *string     `gorm:"type:uuid;default:null"`
	IPAddress  string      `gorm:"column:ip_address;type:varchar(45);not null"`
	UserAgent  string      `gorm:"type:text;not null"`
	User       initialUser `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (initialRefreshToken) TableName() string { return "refresh_tokens" }
