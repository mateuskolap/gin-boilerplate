package port

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	ErrInvalidStoragePath = errors.New("invalid storage path")
	ErrFileExists         = errors.New("storage file already exists")
	ErrFileNotFound       = errors.New("storage file not found")
	ErrFileTooLarge       = errors.New("storage file exceeds size limit")
	ErrNotRegularFile     = errors.New("storage path is not a regular file")
)

type FileInfo struct {
	Size       int64
	ModifiedAt time.Time
}

type Storage interface {
	Put(ctx context.Context, path string, src io.Reader) error
	Open(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) (bool, error)
	Stat(ctx context.Context, path string) (FileInfo, error)
}
