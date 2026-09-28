package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gin-boilerplate/internal/domain/port"
)

func newTestLocal(t *testing.T) (*Local, string) {
	t.Helper()
	rootPath := filepath.Join(t.TempDir(), "objects")
	store, err := NewLocal(rootPath)
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, rootPath
}

func TestLocalStoragePutOpenAndDelete(t *testing.T) {
	store, _ := newTestLocal(t)
	ctx := context.Background()
	key := "users/42/avatar.png"
	if err := store.Put(ctx, key, strings.NewReader("image-data"), port.PutOptions{MaxBytes: 20}); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	file, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	data, err := io.ReadAll(file.Content)
	closeErr := file.Content.Close()
	if err != nil || closeErr != nil || string(data) != "image-data" || file.Size != int64(len(data)) {
		t.Fatalf("Open() content=%q size=%d read error=%v close error=%v", data, file.Size, err, closeErr)
	}
	if err := store.Put(ctx, key, strings.NewReader("replacement"), port.PutOptions{MaxBytes: 20}); !errors.Is(err, port.ErrFileExists) {
		t.Fatalf("Put() existing key error = %v, want ErrFileExists", err)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete() missing key error = %v, want nil", err)
	}
	if _, err := store.Open(ctx, key); !errors.Is(err, port.ErrFileNotFound) {
		t.Fatalf("Open() missing key error = %v, want ErrFileNotFound", err)
	}
}

func TestLocalStorageRejectsUnsafeKeysAndInvalidLimits(t *testing.T) {
	store, _ := newTestLocal(t)
	for _, key := range []string{"", ".", "../outside", "/absolute", "users\\avatar", "users/.storage-tmp-hidden"} {
		if err := store.Put(context.Background(), key, strings.NewReader("x"), port.PutOptions{MaxBytes: 10}); !errors.Is(err, port.ErrInvalidStoragePath) {
			t.Errorf("Put(%q) error = %v, want ErrInvalidStoragePath", key, err)
		}
	}
	for _, maxBytes := range []int64{0, -1, int64(^uint64(0) >> 1)} {
		if err := store.Put(context.Background(), "valid/key", strings.NewReader("x"), port.PutOptions{MaxBytes: maxBytes}); !errors.Is(err, port.ErrInvalidFileSizeLimit) {
			t.Errorf("Put() max bytes %d error = %v, want ErrInvalidFileSizeLimit", maxBytes, err)
		}
	}
	if err := store.Put(context.Background(), "valid/key", nil, port.PutOptions{MaxBytes: 10}); err == nil {
		t.Fatal("Put() accepted nil content")
	}
	if err := store.Put(context.Background(), "valid/key", strings.NewReader("too long"), port.PutOptions{MaxBytes: 3}); !errors.Is(err, port.ErrFileTooLarge) {
		t.Fatalf("Put() oversize error = %v, want ErrFileTooLarge", err)
	}
}

func TestLocalStorageHonorsCanceledContextAndRegularFiles(t *testing.T) {
	store, rootPath := newTestLocal(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.Put(ctx, "cancelled/file", strings.NewReader("x"), port.PutOptions{MaxBytes: 10}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Put() with canceled context error = %v", err)
	}
	if _, err := store.Open(ctx, "cancelled/file"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Open() with canceled context error = %v", err)
	}
	if err := store.Delete(ctx, "cancelled/file"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete() with canceled context error = %v", err)
	}

	if err := os.Mkdir(filepath.Join(rootPath, "directory"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open(context.Background(), "directory"); !errors.Is(err, port.ErrNotRegularFile) {
		t.Fatalf("Open() directory error = %v, want ErrNotRegularFile", err)
	}
	if err := store.Delete(context.Background(), "directory"); !errors.Is(err, port.ErrNotRegularFile) {
		t.Fatalf("Delete() directory error = %v, want ErrNotRegularFile", err)
	}

	externalFile := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(externalFile, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(externalFile, filepath.Join(rootPath, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open(context.Background(), "link"); !errors.Is(err, port.ErrNotRegularFile) {
		t.Fatalf("Open() symlink error = %v, want ErrNotRegularFile", err)
	}
}
