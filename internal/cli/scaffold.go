package cli

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type generatedFile struct {
	path    string
	content []byte
}

func makeDomain(root, name string) (string, error) {
	return makeDomainWithOptions(root, name, false, false)
}

func makeDomainWithSoftDelete(root, name string, softDelete bool) (string, error) {
	return makeDomainWithOptions(root, name, softDelete, false)
}

func makeDomainWithOptions(root, name string, softDelete, activityLogs bool) (string, error) {
	entity, stem, fileStem, feature, err := domainNames(name)
	if err != nil {
		return "", err
	}

	domainBaseModel := "shared.BaseModel"
	modelImports := fmt.Sprintf(
		"postgresinfra \"gin-boilerplate/internal/infra/postgres\"\n\t%sdomain \"gin-boilerplate/internal/%s/domain\"",
		stem,
		feature,
	)
	modelFields := "postgresinfra.BaseModel"
	modelFromDomainBody := ""
	domainFromModelBody := ""
	domainImports := ""
	activityLogMethods := ""
	activityLogGuidance := ""
	if softDelete {
		domainBaseModel = "shared.BaseSoftDeleteModel"
		modelImports += "\n\t\"gorm.io/gorm\""
		modelFields = "postgresinfra.BaseModel\n\tDeletedAt gorm.DeletedAt `gorm:\"index\"`"
		modelFromDomainBody = `
	if entity.DeletedAt != nil {
		model.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	}`
		domainFromModelBody = `
	if model.DeletedAt.Valid {
		deletedAt := model.DeletedAt.Time
		entity.DeletedAt = &deletedAt
	}`
	}
	if activityLogs {
		domainImports = `activitylogdomain "gin-boilerplate/internal/activity_logs/domain"`
		activityLogGuidance = "// Mark fields with `activity:\"track\"` to include their values in activity logs."
		activityLogMethods = fmt.Sprintf(`
func (e %s) ActivityLogSubjectType() activitylogdomain.ActivitySubjectType {
	return activitylogdomain.ActivitySubjectType(%q)
}

func (e %s) ActivityLogID() uuid.UUID { return e.ID }
`, entity, fileStem, entity)
	}

	activityLogRepositoryImport := ""
	repositoryType := fmt.Sprintf("*postgresinfra.BaseRepository[%s, *%sModel]", stem+"domain."+entity, entity)
	repositoryArgs := "db *gorm.DB"
	baseRepository := fmt.Sprintf(`postgresinfra.NewBaseRepository(db,
			func() *%sModel { return &%sModel{} },
			func(entity *%s) *%sModel { return %sModelFromDomain(entity) },
			%sDomainFromModel,
		)`, entity, entity, stem+"domain."+entity, entity, stem, stem)
	repositoryInitializer := "BaseRepository: " + baseRepository
	if activityLogs {
		activityLogRepositoryImport = `activitylogdomain "gin-boilerplate/internal/activity_logs/domain"`
		repositoryType = fmt.Sprintf("*postgresinfra.ActivityLoggingRepository[%s]", stem+"domain."+entity)
		repositoryArgs += ", activityLogRepo activitylogdomain.ActivityLogRepository"
		repositoryInitializer = fmt.Sprintf("ActivityLoggingRepository: postgresinfra.NewActivityLoggingRepository(db, %s, activityLogRepo)", baseRepository)
	}

	replacer := strings.NewReplacer(
		"{{Entity}}", entity,
		"{{Stem}}", stem,
		"{{Feature}}", feature,
		"{{DomainBaseModel}}", domainBaseModel,
		"{{DomainImports}}", domainImports,
		"{{ActivityLogGuidance}}", activityLogGuidance,
		"{{ActivityLogMethods}}", activityLogMethods,
		"{{ModelImports}}", modelImports,
		"{{ModelFields}}", modelFields,
		"{{ModelFromDomainBody}}", modelFromDomainBody,
		"{{DomainFromModelBody}}", domainFromModelBody,
		"{{ActivityLogRepositoryImport}}", activityLogRepositoryImport,
		"{{RepositoryType}}", repositoryType,
		"{{RepositoryArgs}}", repositoryArgs,
		"{{RepositoryInitializer}}", repositoryInitializer,
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
	sql := domainMigrationSQL(feature, softDelete)
	if err := createMigrationWithContent(
		filepath.Join(root, migrationsDirectory),
		time.Now().UTC(),
		"create_"+feature,
		sql,
	); err != nil {
		return "", fmt.Errorf("create domain migration: %w", err)
	}
	complete = true
	return feature, nil
}

func domainMigrationSQL(feature string, softDelete bool) migrationSQL {
	columns := []string{
		"id UUID PRIMARY KEY DEFAULT uuidv7()",
		"created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP",
		"updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP",
	}
	if softDelete {
		columns = append(columns, "deleted_at TIMESTAMPTZ")
	}

	upSQL := fmt.Sprintf("CREATE TABLE %s (\n    %s\n);\n", feature, strings.Join(columns, ",\n    "))
	if softDelete {
		upSQL += fmt.Sprintf("\nCREATE INDEX idx_%s_deleted_at ON %s (deleted_at);\n", feature, feature)
	}
	return migrationSQL{up: upSQL, down: fmt.Sprintf("DROP TABLE %s;\n", feature)}
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
	{{DomainImports}}
	"gin-boilerplate/internal/domain/shared"
	"uuid"
)

type {{Entity}} struct {
	{{DomainBaseModel}}
	{{ActivityLogGuidance}}
}

{{ActivityLogMethods}}

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
		BaseFindUseCase: sharedapp.NewBaseFindUseCase(repository),
		BaseDeleteUseCase: sharedapp.NewBaseDeleteUseCase(repository),
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
	{{ModelImports}}
)

type {{Entity}}Model struct {
	{{ModelFields}}
}

func ({{Entity}}Model) TableName() string { return "{{Feature}}" }

func {{Stem}}ModelFromDomain(entity *{{Stem}}domain.{{Entity}}) *{{Entity}}Model {
	model := &{{Entity}}Model{
		BaseModel: postgresinfra.BaseModelFromDomain(entity.BaseModel),
	}
	{{ModelFromDomainBody}}
	return model
}

func {{Stem}}DomainFromModel(model *{{Entity}}Model) *{{Stem}}domain.{{Entity}} {
	entity := &{{Stem}}domain.{{Entity}}{
		BaseModel: model.BaseModel.ToDomain(),
	}
	{{DomainFromModelBody}}
	return entity
}
`

const repositorySource = `package postgres

import (
	{{ActivityLogRepositoryImport}}
	postgresinfra "gin-boilerplate/internal/infra/postgres"
	{{Stem}}domain "gin-boilerplate/internal/{{Feature}}/domain"
	"gorm.io/gorm"
)

type {{Stem}}Repository struct {
	{{RepositoryType}}
}

var _ {{Stem}}domain.{{Entity}}Repository = (*{{Stem}}Repository)(nil)

func New{{Entity}}Repository({{RepositoryArgs}}) {{Stem}}domain.{{Entity}}Repository {
	return &{{Stem}}Repository{
		{{RepositoryInitializer}},
	}
}
`
