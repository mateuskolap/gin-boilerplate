package cli

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
)

type generatedFile struct {
	path    string
	content []byte
}

func makeDomain(root, name string) (string, error) {
	entity, stem, fileStem, feature, err := domainNames(name)
	if err != nil {
		return "", err
	}

	replacer := strings.NewReplacer(
		"{{Entity}}", entity,
		"{{Stem}}", stem,
		"{{Feature}}", feature,
	)
	files := []generatedFile{
		{filepath.Join("domain", fileStem+".go"), []byte(domainSource)},
		{filepath.Join("application", fileStem+".go"), []byte(applicationSource)},
		{filepath.Join("adapters", "postgres", fileStem+"_model.go"), []byte(modelSource)},
		{filepath.Join("adapters", "postgres", fileStem+"_repository.go"), []byte(repositorySource)},
	}
	for i := range files {
		formatted, err := format.Source([]byte(replacer.Replace(string(files[i].content))))
		if err != nil {
			return "", fmt.Errorf("format generated %s: %w", files[i].path, err)
		}
		files[i].content = formatted
	}

	featureRoot := filepath.Join(root, "internal", feature)
	if err := os.Mkdir(featureRoot, 0o755); err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("domain %q already exists at internal/%s", name, feature)
		}
		return "", fmt.Errorf("create domain directory: %w", err)
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(featureRoot)
		}
	}()
	for _, dir := range []string{"domain", "application", filepath.Join("adapters", "postgres")} {
		path := filepath.Join(featureRoot, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return "", fmt.Errorf("create %s: %w", path, err)
		}
	}

	for _, file := range files {
		path := filepath.Join(featureRoot, file.path)
		if err := os.WriteFile(path, file.content, 0o644); err != nil {
			return "", fmt.Errorf("write %s: %w", path, err)
		}
	}
	complete = true
	return feature, nil
}

func domainNames(name string) (entity, stem, fileStem, feature string, err error) {
	if len(name) == 0 || len(name) > 64 || name[0] < 'A' || name[0] > 'Z' {
		return "", "", "", "", fmt.Errorf("domain name must be PascalCase, e.g. User or ActivityLog")
	}
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !isUpper(c) && !isLower(c) && !isDigit(c) {
			return "", "", "", "", fmt.Errorf("domain name must be PascalCase, e.g. User or ActivityLog")
		}
	}

	snake := toSnakeCase(name)
	return name, lowerCamelCase(name), snake, pluralize(snake), nil
}

func toSnakeCase(name string) string {
	var result strings.Builder
	for i := 0; i < len(name); i++ {
		c := name[i]
		if i > 0 && isUpper(c) && (isLower(name[i-1]) || isDigit(name[i-1]) || (isUpper(name[i-1]) && i+1 < len(name) && isLower(name[i+1]))) {
			result.WriteByte('_')
		}
		if isUpper(c) {
			c += 'a' - 'A'
		}
		result.WriteByte(c)
	}
	return result.String()
}

func lowerCamelCase(name string) string {
	end := 1
	for end < len(name) && isUpper(name[end]) {
		if end+1 < len(name) && isLower(name[end+1]) {
			break
		}
		end++
	}
	return strings.ToLower(name[:end]) + name[end:]
}

func pluralize(snake string) string {
	lastWordStart := strings.LastIndexByte(snake, '_') + 1
	lastWord := snake[lastWordStart:]
	// ponytail: regular English plural rules only; add an explicit mapping if an irregular domain name is introduced.
	switch {
	case strings.HasSuffix(lastWord, "ch"), strings.HasSuffix(lastWord, "sh"), strings.HasSuffix(lastWord, "s"), strings.HasSuffix(lastWord, "x"), strings.HasSuffix(lastWord, "z"):
		return snake + "es"
	case strings.HasSuffix(lastWord, "y") && len(lastWord) > 1 && !strings.ContainsRune("aeiou", rune(lastWord[len(lastWord)-2])):
		return snake[:len(snake)-1] + "ies"
	default:
		return snake + "s"
	}
}

func isUpper(c byte) bool { return c >= 'A' && c <= 'Z' }
func isLower(c byte) bool { return c >= 'a' && c <= 'z' }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }

const domainSource = `package domain

import (
	"context"
	"gin-boilerplate/internal/domain/shared"
	"uuid"
)

type {{Entity}} struct {
	shared.BaseModel
}

type {{Entity}}Repository interface {
	Create(context.Context, *{{Entity}}) error
	GetByID(context.Context, uuid.UUID) (*{{Entity}}, error)
	Update(context.Context, *{{Entity}}) error
	Delete(context.Context, uuid.UUID) error
	List(context.Context, shared.PaginationParams, []shared.Filter) (*shared.PaginatedResult[{{Entity}}], error)
}

type {{Entity}}UseCase interface {
	shared.BaseListUseCase[{{Entity}}]
	shared.BaseFindUseCase[{{Entity}}]
	shared.BaseDeleteUseCase
	Create(context.Context, *{{Entity}}) error
	Update(context.Context, *{{Entity}}) error
}
`

const applicationSource = `package application

import (
	"context"
	sharedapp "gin-boilerplate/internal/application"
	"gin-boilerplate/internal/domain/shared"
	{{Stem}}domain "gin-boilerplate/internal/{{Feature}}/domain"
)

var allowed{{Entity}}FilterFields = map[string]bool{
	"id": true, "created_at": true, "updated_at": true,
}

type {{Stem}}UseCase struct {
	shared.BaseListUseCase[{{Stem}}domain.{{Entity}}]
	shared.BaseFindUseCase[{{Stem}}domain.{{Entity}}]
	shared.BaseDeleteUseCase
	repository {{Stem}}domain.{{Entity}}Repository
}

func New{{Entity}}UseCase(repository {{Stem}}domain.{{Entity}}Repository) {{Stem}}domain.{{Entity}}UseCase {
	return &{{Stem}}UseCase{
		BaseListUseCase: sharedapp.NewBaseListUseCase(repository, allowed{{Entity}}FilterFields),
		BaseFindUseCase: sharedapp.NewBaseFindUseCase[{{Stem}}domain.{{Entity}}](repository),
		BaseDeleteUseCase: sharedapp.NewBaseDeleteUseCase[{{Stem}}domain.{{Entity}}](repository),
		repository: repository,
	}
}

func (u *{{Stem}}UseCase) Create(ctx context.Context, entity *{{Stem}}domain.{{Entity}}) error {
	return u.repository.Create(ctx, entity)
}

func (u *{{Stem}}UseCase) Update(ctx context.Context, entity *{{Stem}}domain.{{Entity}}) error {
	return u.repository.Update(ctx, entity)
}
`

const modelSource = `package postgres

import (
	postgresinfra "gin-boilerplate/internal/infra/postgres"
	{{Stem}}domain "gin-boilerplate/internal/{{Feature}}/domain"
)

type {{Entity}}Model struct {
	postgresinfra.BaseModel
}

func ({{Entity}}Model) TableName() string { return "{{Feature}}" }

func {{Stem}}ModelFromDomain(entity *{{Stem}}domain.{{Entity}}) *{{Entity}}Model {
	return &{{Entity}}Model{
		ID: entity.ID, CreatedAt: entity.CreatedAt, UpdatedAt: entity.UpdatedAt,
	}
}

func {{Stem}}DomainFromModel(model any) *{{Stem}}domain.{{Entity}} {
	entity := model.(*{{Entity}}Model)
	return &{{Stem}}domain.{{Entity}}{
		ID: entity.ID, CreatedAt: entity.CreatedAt, UpdatedAt: entity.UpdatedAt,
	}
}
`

const repositorySource = `package postgres

import (
	postgresinfra "gin-boilerplate/internal/infra/postgres"
	{{Stem}}domain "gin-boilerplate/internal/{{Feature}}/domain"
	"gorm.io/gorm"
)

type {{Stem}}Repository struct {
	*postgresinfra.BaseRepository[{{Stem}}domain.{{Entity}}]
}

var _ {{Stem}}domain.{{Entity}}Repository = (*{{Stem}}Repository)(nil)

func New{{Entity}}Repository(db *gorm.DB) {{Stem}}domain.{{Entity}}Repository {
	return &{{Stem}}Repository{
		BaseRepository: postgresinfra.NewBaseRepository[{{Stem}}domain.{{Entity}}](db,
			func() any { return &{{Entity}}Model{} },
			func(entity *{{Stem}}domain.{{Entity}}) any { return {{Stem}}ModelFromDomain(entity) },
			{{Stem}}DomainFromModel,
		),
	}
}
`
