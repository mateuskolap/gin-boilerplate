package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gin-boilerplate/internal/domain/port"
)

const temporaryPrefix = ".storage-tmp-"

type Local struct {
	root    *os.Root
	maxSize int64
}

var _ port.Storage = (*Local)(nil)

func NewLocal(rootPath string, maxSize int64) (*Local, error) {
	if strings.TrimSpace(rootPath) == "" {
		return nil, errors.New("storage root must not be empty")
	}
	if maxSize <= 0 || maxSize == math.MaxInt64 {
		return nil, errors.New("storage size limit must be between 1 and MaxInt64-1")
	}

	absoluteRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("resolve storage root: %w", err)
	}
	if err := os.MkdirAll(absoluteRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	root, err := os.OpenRoot(absoluteRoot)
	if err != nil {
		return nil, fmt.Errorf("open storage root: %w", err)
	}
	store := &Local{root: root, maxSize: maxSize}
	temporaryKey, temporaryFile, err := store.createTemporary(".")
	if err != nil {
		_ = root.Close()
		return nil, fmt.Errorf("check storage write access: %w", err)
	}
	if err := temporaryFile.Close(); err != nil {
		_ = root.Remove(temporaryKey)
		_ = root.Close()
		return nil, fmt.Errorf("close storage write check: %w", err)
	}
	if err := root.Remove(temporaryKey); err != nil {
		_ = root.Close()
		return nil, fmt.Errorf("clean up storage write check: %w", err)
	}
	return store, nil
}

func (s *Local) Close() error {
	return s.root.Close()
}

func (s *Local) Put(ctx context.Context, key string, src io.Reader) (resultErr error) {
	if err := validateKey(key); err != nil {
		return err
	}
	if src == nil {
		return errors.New("storage source must not be nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	directory := path.Dir(key)
	if err := s.root.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create storage directory: %w", err)
	}
	temporaryKey, temporaryFile, err := s.createTemporary(directory)
	if err != nil {
		return err
	}
	defer func() {
		_ = temporaryFile.Close()
		if err := s.root.Remove(temporaryKey); err != nil {
			if resultErr == nil {
				slog.Warn("failed to remove storage temporary file", "path", temporaryKey, "error", err)
			} else {
				resultErr = errors.Join(resultErr, fmt.Errorf("remove storage temporary file: %w", err))
			}
		}
	}()

	limitedSource := io.LimitReader(&contextReader{ctx: ctx, reader: src}, s.maxSize+1)
	written, err := io.Copy(temporaryFile, limitedSource)
	if written > s.maxSize {
		return port.ErrFileTooLarge
	}
	if err != nil {
		return fmt.Errorf("write storage file: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := temporaryFile.Sync(); err != nil {
		return fmt.Errorf("sync storage file: %w", err)
	}
	if err := temporaryFile.Close(); err != nil {
		return fmt.Errorf("close storage file: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.root.Link(temporaryKey, key); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return port.ErrFileExists
		}
		return fmt.Errorf("publish storage file: %w", err)
	}
	return nil
}

func (s *Local) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := s.fileInfo(key); err != nil {
		return nil, err
	}

	file, err := s.root.Open(key)
	if err != nil {
		return nil, fileError("open storage file", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("inspect open storage file: %w", err)
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, port.ErrNotRegularFile
	}
	if err := ctx.Err(); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func (s *Local) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := s.fileInfo(key); err != nil {
		if errors.Is(err, port.ErrFileNotFound) {
			return nil
		}
		return err
	}
	if err := s.root.Remove(key); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("delete storage file: %w", err)
	}
	return nil
}

func (s *Local) Exists(ctx context.Context, key string) (bool, error) {
	if err := validateKey(key); err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if _, err := s.fileInfo(key); err != nil {
		if errors.Is(err, port.ErrFileNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Local) Stat(ctx context.Context, key string) (port.FileInfo, error) {
	if err := validateKey(key); err != nil {
		return port.FileInfo{}, err
	}
	if err := ctx.Err(); err != nil {
		return port.FileInfo{}, err
	}
	info, err := s.fileInfo(key)
	if err != nil {
		return port.FileInfo{}, err
	}
	return port.FileInfo{Size: info.Size(), ModifiedAt: info.ModTime().UTC()}, nil
}

func (s *Local) fileInfo(key string) (fs.FileInfo, error) {
	info, err := s.root.Lstat(key)
	if err != nil {
		return nil, fileError("inspect storage file", err)
	}
	if !info.Mode().IsRegular() {
		return nil, port.ErrNotRegularFile
	}
	return info, nil
}

func (s *Local) createTemporary(directory string) (string, *os.File, error) {
	for range 5 {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return "", nil, fmt.Errorf("generate storage temporary name: %w", err)
		}
		key := path.Join(directory, temporaryPrefix+hex.EncodeToString(random[:]))
		file, err := s.root.OpenFile(key, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", nil, fmt.Errorf("create storage temporary file: %w", err)
		}
		return key, file, nil
	}
	return "", nil, errors.New("could not create a unique storage temporary file")
}

func validateKey(key string) error {
	if !fs.ValidPath(key) || key == "." || strings.ContainsAny(key, "\\\x00") {
		return port.ErrInvalidStoragePath
	}
	for segment := range strings.SplitSeq(key, "/") {
		if strings.HasPrefix(segment, temporaryPrefix) {
			return port.ErrInvalidStoragePath
		}
	}
	return nil
}

func fileError(operation string, err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s: %w: %w", operation, port.ErrFileNotFound, err)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}
