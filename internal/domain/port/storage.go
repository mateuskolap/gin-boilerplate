package port

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	// ErrInvalidStoragePath means the storage key is invalid for the implementation.
	ErrInvalidStoragePath = errors.New("invalid storage path")
	// ErrFileExists means Put cannot create the key because an object already exists there.
	ErrFileExists = errors.New("storage file already exists")
	// ErrFileNotFound means the requested object does not exist.
	ErrFileNotFound = errors.New("storage file not found")
	// ErrFileTooLarge means Put exceeded the supplied size limit.
	ErrFileTooLarge = errors.New("storage file exceeds size limit")
	// ErrInvalidFileSizeLimit means Put received a non-positive or unsupported size limit.
	ErrInvalidFileSizeLimit = errors.New("invalid storage file size limit")
	// ErrNotRegularFile is returned by filesystem-backed storage when a key resolves to a
	// directory or another non-regular filesystem entry. Object stores may not return it.
	ErrNotRegularFile = errors.New("storage path is not a regular file")
)

type OpenedFile struct {
	Content io.ReadCloser
	Info    FileInfo
}

type FileInfo struct {
	Size       int64
	ModifiedAt time.Time
}

type PutOptions struct {
	// MaxBytes is the maximum number of bytes Put may read from src.
	MaxBytes int64
}

type Storage interface {
	// Put stores src under a key that does not already exist. It returns
	// ErrInvalidStoragePath for an invalid key, ErrFileExists when the key is taken,
	// ErrInvalidFileSizeLimit when options.MaxBytes is invalid, and ErrFileTooLarge
	// when the limit is exceeded. src must not be nil. It may also return ctx.Err()
	// or an implementation-specific read/write error.
	Put(ctx context.Context, path string, src io.Reader, options PutOptions) error

	// Open opens an existing object for reading. The caller must close the returned
	// reader. It returns ErrInvalidStoragePath for an invalid key and ErrFileNotFound
	// when no object exists. Filesystem-backed implementations may also return
	// ErrNotRegularFile. It may return ctx.Err() or an implementation-specific I/O error.
	Open(ctx context.Context, path string) (OpenedFile, error)

	// Delete removes an object. Deleting a missing key succeeds and returns nil. It
	// returns ErrInvalidStoragePath for an invalid key. Filesystem-backed
	// implementations may also return ErrNotRegularFile. It may return ctx.Err() or
	// an implementation-specific I/O error.
	Delete(ctx context.Context, path string) error

	// Exists reports whether an object exists. A missing key returns (false, nil).
	// It returns ErrInvalidStoragePath for an invalid key. Filesystem-backed
	// implementations may also return ErrNotRegularFile. It may return ctx.Err() or
	// an implementation-specific I/O error.
	Exists(ctx context.Context, path string) (bool, error)

	// Stat returns metadata for an existing object. It returns ErrInvalidStoragePath
	// for an invalid key and ErrFileNotFound when no object exists. Filesystem-backed
	// implementations may also return ErrNotRegularFile. It may return ctx.Err() or
	// an implementation-specific I/O error.
	Stat(ctx context.Context, path string) (FileInfo, error)
}
