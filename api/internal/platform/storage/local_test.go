package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genkovich/task-tracker/api/internal/platform/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verify Local satisfies the interface at compile time.
var _ storage.ObjectStorage = (*storage.Local)(nil)

func TestLocalStorage_Put_WritesFileAndReturnsURL(t *testing.T) {
	baseDir := t.TempDir()
	const publicBase = "http://localhost/files"
	const key = "avatars/user1.png"
	const content = "image-bytes"

	s := storage.NewLocal(baseDir, publicBase)
	url, err := s.Put(context.Background(), storage.Object{
		Key:         key,
		ContentType: "image/png",
		Body:        strings.NewReader(content),
		Size:        int64(len(content)),
	})

	require.NoError(t, err)
	assert.Equal(t, "http://localhost/files/avatars/user1.png", url)

	written, err := os.ReadFile(filepath.Join(baseDir, key))
	require.NoError(t, err)
	assert.Equal(t, content, string(written))
}

func TestLocalStorage_Put_CreatesNestedDirs(t *testing.T) {
	baseDir := t.TempDir()
	s := storage.NewLocal(baseDir, "http://localhost/files")

	_, err := s.Put(context.Background(), storage.Object{
		Key:         "a/b/c/file.png",
		ContentType: "image/png",
		Body:        strings.NewReader("x"),
		Size:        1,
	})

	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(baseDir, "a", "b", "c", "file.png"))
	require.NoError(t, err)
	assert.False(t, info.IsDir())
}

func TestLocalStorage_Put_PreservesBytes(t *testing.T) {
	baseDir := t.TempDir()
	s := storage.NewLocal(baseDir, "http://localhost/files")
	const payload = "binary\x00data\xff"

	_, err := s.Put(context.Background(), storage.Object{
		Key:         "avatars/bin.bin",
		ContentType: "application/octet-stream",
		Body:        strings.NewReader(payload),
		Size:        int64(len(payload)),
	})

	require.NoError(t, err)
	got, err := os.ReadFile(filepath.Join(baseDir, "avatars", "bin.bin"))
	require.NoError(t, err)
	assert.Equal(t, payload, string(got))
}

func TestLocalStorage_Delete_RemovesFile(t *testing.T) {
	baseDir := t.TempDir()
	s := storage.NewLocal(baseDir, "http://localhost/files")

	// Create file first via Put
	_, err := s.Put(context.Background(), storage.Object{
		Key:         "avatars/todelete.png",
		ContentType: "image/png",
		Body:        strings.NewReader("data"),
		Size:        4,
	})
	require.NoError(t, err)

	// Confirm it exists
	_, err = os.Stat(filepath.Join(baseDir, "avatars", "todelete.png"))
	require.NoError(t, err, "file must exist before delete")

	err = s.Delete(context.Background(), "avatars/todelete.png")
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(baseDir, "avatars", "todelete.png"))
	assert.True(t, os.IsNotExist(err), "file must be gone after delete")
}

func TestLocalStorage_Delete_MissingFile_IsNotError(t *testing.T) {
	baseDir := t.TempDir()
	s := storage.NewLocal(baseDir, "http://localhost/files")

	err := s.Delete(context.Background(), "avatars/nonexistent.png")
	assert.NoError(t, err, "deleting a non-existent file must be a no-op")
}

func TestLocalStorage_Put_RejectsPathTraversal(t *testing.T) {
	baseDir := t.TempDir()
	s := storage.NewLocal(baseDir, "http://localhost/files")

	_, err := s.Put(context.Background(), storage.Object{
		Key:         "../etc/passwd",
		ContentType: "text/plain",
		Body:        strings.NewReader("evil"),
		Size:        4,
	})

	require.Error(t, err, "path traversal key must be rejected")
	assert.Contains(t, err.Error(), "invalid key")
}

func TestLocalStorage_Put_RejectsAbsoluteKey(t *testing.T) {
	baseDir := t.TempDir()
	s := storage.NewLocal(baseDir, "http://localhost/files")

	_, err := s.Put(context.Background(), storage.Object{
		Key:         "/etc/passwd",
		ContentType: "text/plain",
		Body:        strings.NewReader("evil"),
		Size:        4,
	})

	require.Error(t, err, "absolute key must be rejected")
	assert.Contains(t, err.Error(), "invalid key")
}
