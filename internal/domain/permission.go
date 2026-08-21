package domain

import "context"

type Permission struct {
	BaseModel
	Name string `json:"name" gorm:"unique;not null"`
}

type PermissionRepository interface {
	BaseRepository[Permission]
	UpsertByName(ctx context.Context, permissions []Permission) error
	DeleteByNames(ctx context.Context, names []string) error
}

type PermissionUseCase interface {
	SeedPermissions(ctx context.Context) error
}
