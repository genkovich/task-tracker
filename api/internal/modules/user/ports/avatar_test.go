package ports

// Unit tests for the avatar upload/delete handlers.
// These tests run without a real database by using a stub service and a
// fake ObjectStorage, so they execute with `go test ./...` (no build tags).

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/user/app"
	"github.com/genkovich/task-tracker/api/internal/modules/user/domain"
	"github.com/genkovich/task-tracker/api/internal/platform/authmw"
	"github.com/genkovich/task-tracker/api/internal/platform/storage"
)

// Magic bytes that http.DetectContentType recognises for each format.
var imageMagic = map[string][]byte{
	"image/png":  {0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
	"image/jpeg": {0xFF, 0xD8, 0xFF, 0xE0},
	"image/webp": append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), make([]byte, 16)...),
}

// ---------------------------------------------------------------------------
// fakeStorage — test double for storage.ObjectStorage
// ---------------------------------------------------------------------------

type fakeStorage struct {
	putCalls    []storage.Object
	deleteCalls []string
	putURL      string
	putErr      error
	deleteErr   error
}

func (f *fakeStorage) Put(_ context.Context, obj storage.Object) (string, error) {
	f.putCalls = append(f.putCalls, obj)
	if f.putErr != nil {
		return "", f.putErr
	}
	return f.putURL, nil
}

func (f *fakeStorage) Delete(_ context.Context, key string) error {
	f.deleteCalls = append(f.deleteCalls, key)
	return f.deleteErr
}

// PresignPut, PresignGet, and Head are unused by the avatar handler under
// test; these stubs exist only to satisfy storage.ObjectStorage.
func (f *fakeStorage) PresignPut(_ context.Context, _, _ string, _ int64, _ time.Duration) (string, error) {
	return "", nil
}

func (f *fakeStorage) PresignGet(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}

func (f *fakeStorage) Head(_ context.Context, _ string) (int64, bool, error) {
	return 0, false, nil
}

// ---------------------------------------------------------------------------
// fakeUserRepo — test double for domain.UserRepository
// ---------------------------------------------------------------------------

type fakeUserRepo struct {
	users map[uuid.UUID]*domain.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: make(map[uuid.UUID]*domain.User)}
}

func (r *fakeUserRepo) Create(_ context.Context, user *domain.User) error {
	r.users[user.ID] = user
	return nil
}

func (r *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeUserRepo) List(_ context.Context, _ domain.ListParams) (*domain.ListResult, error) {
	return &domain.ListResult{}, nil
}

func (r *fakeUserRepo) Update(_ context.Context, user *domain.User) error {
	r.users[user.ID] = user
	return nil
}

func (r *fakeUserRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.users, id)
	return nil
}

// ---------------------------------------------------------------------------
// test helpers
// ---------------------------------------------------------------------------

// newTestHandler builds a Handler with a seeded user and returns the handler,
// fake storage, and the seeded user ID.
func newTestHandler(t *testing.T, storageOverride *fakeStorage) (*Handler, *fakeStorage, uuid.UUID) {
	t.Helper()

	userID := uuid.MustParse("01960000-0000-7000-8000-000000000001")
	user := &domain.User{
		ID:    userID,
		Email: "test@example.com",
		Role:  "member",
	}

	repo := newFakeUserRepo()
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svc := app.NewService(repo)

	fs := storageOverride
	if fs == nil {
		fs = &fakeStorage{putURL: "/files/avatars/" + userID.String() + ".png"}
	}

	h := &Handler{service: svc, storage: fs}
	return h, fs, userID
}

// buildMultipartRequest constructs a multipart/form-data POST request with
// a synthetic image file in the "avatar" field.
func buildMultipartRequest(t *testing.T, userID uuid.UUID, filename, contentType string, bodySize int) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	// Create the file part with explicit Content-Type via a MIME header.
	mh := make(textproto.MIMEHeader)
	mh.Set("Content-Disposition", fmt.Sprintf(`form-data; name="avatar"; filename="%s"`, filename))
	mh.Set("Content-Type", contentType)

	part, err := mw.CreatePart(mh)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}

	payload := buildImagePayload(contentType, bodySize)
	if _, err := part.Write(payload); err != nil {
		t.Fatalf("write part: %v", err)
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/avatar", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	// Inject auth claims so the handler can extract the user ID.
	ctx := authmw.WithClaims(req.Context(), &authmw.Claims{
		UserID: userID,
		Email:  "test@example.com",
		Role:   "member",
	})
	return req.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestUploadAvatar_StoresViaStorage_ReturnsUpdatedURL verifies that on a
// valid PNG upload the handler calls storage.Put with the correct key and
// content-type, and returns the public URL (with cache-buster) in the
// response avatar_url field.
func TestUploadAvatar_StoresViaStorage_ReturnsUpdatedURL(t *testing.T) {
	fs := &fakeStorage{putURL: "https://cdn.example.com/avatars/01960000-0000-7000-8000-000000000001.png"}
	h, _, userID := newTestHandler(t, fs)

	req := buildMultipartRequest(t, userID, "photo.png", "image/png", 512)
	w := httptest.NewRecorder()
	h.handleUploadAvatar(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// storage.Put must have been called exactly once.
	if len(fs.putCalls) != 1 {
		t.Fatalf("Put call count: got %d, want 1", len(fs.putCalls))
	}

	wantKey := "avatars/" + userID.String() + ".png"
	if fs.putCalls[0].Key != wantKey {
		t.Errorf("Put key: got %q, want %q", fs.putCalls[0].Key, wantKey)
	}
	if fs.putCalls[0].ContentType != "image/png" {
		t.Errorf("Put content-type: got %q, want %q", fs.putCalls[0].ContentType, "image/png")
	}

	// Response must contain avatar_url with cache-buster.
	body := w.Body.String()
	if !strings.Contains(body, "avatar_url") {
		t.Errorf("response missing avatar_url field: %s", body)
	}
	if !strings.Contains(body, "?v=") {
		t.Errorf("response avatar_url missing cache-buster ?v=: %s", body)
	}
}

// TestUploadAvatar_CleansUpOtherExtensions_AfterPut verifies the handler
// uploads first (so the new avatar is live before we touch anything else),
// then deletes the other two extensions to clean up stale variants. The
// extension matching the new upload is NOT deleted — we just wrote to it.
func TestUploadAvatar_CleansUpOtherExtensions_AfterPut(t *testing.T) {
	fs := &fakeStorage{putURL: "/files/avatars/x.png"}
	h, _, userID := newTestHandler(t, fs)

	req := buildMultipartRequest(t, userID, "photo.png", "image/png", 256)
	w := httptest.NewRecorder()
	h.handleUploadAvatar(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	if len(fs.putCalls) == 0 {
		t.Fatal("Put was not called")
	}

	wantDeletes := map[string]bool{
		"avatars/" + userID.String() + ".jpg":  true,
		"avatars/" + userID.String() + ".webp": true,
	}
	// The same-extension key must NOT be deleted.
	disallowed := "avatars/" + userID.String() + ".png"

	got := make(map[string]bool, len(fs.deleteCalls))
	for _, k := range fs.deleteCalls {
		got[k] = true
	}
	for want := range wantDeletes {
		if !got[want] {
			t.Errorf("Delete not called for stale key %q; got: %v", want, fs.deleteCalls)
		}
	}
	if got[disallowed] {
		t.Errorf("Delete was called for the just-uploaded key %q — risk of data loss on partial failure", disallowed)
	}
}

// TestUploadAvatar_RejectsInvalidContentType verifies that uploading an
// unsupported image type (gif) returns 400 and does NOT call storage.Put.
func TestUploadAvatar_RejectsInvalidContentType(t *testing.T) {
	fs := &fakeStorage{}
	h, _, userID := newTestHandler(t, fs)

	req := buildMultipartRequest(t, userID, "anim.gif", "image/gif", 256)
	w := httptest.NewRecorder()
	h.handleUploadAvatar(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	if len(fs.putCalls) != 0 {
		t.Errorf("Put should not have been called for invalid content-type, got %d calls", len(fs.putCalls))
	}
}

// TestUploadAvatar_RejectsOversized verifies that an avatar exceeding 2 MiB
// returns 400 and does NOT call storage.Put.
func TestUploadAvatar_RejectsOversized(t *testing.T) {
	fs := &fakeStorage{}
	h, _, userID := newTestHandler(t, fs)

	// 3 MiB — over the 2 MiB limit.
	req := buildMultipartRequest(t, userID, "big.png", "image/png", 3*1024*1024)
	w := httptest.NewRecorder()
	h.handleUploadAvatar(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	if len(fs.putCalls) != 0 {
		t.Errorf("Put should not have been called for oversized file, got %d calls", len(fs.putCalls))
	}
}

// TestUploadAvatar_StorageError_Returns500_AndDoesNotUpdateDB verifies that
// when storage.Put returns an error the handler responds with an error
// status and the user's avatar_url is NOT updated in the repository.
func TestUploadAvatar_StorageError_Returns500_AndDoesNotUpdateDB(t *testing.T) {
	fs := &fakeStorage{putErr: errors.New("s3 unavailable")}
	h, _, userID := newTestHandler(t, fs)

	req := buildMultipartRequest(t, userID, "photo.png", "image/png", 256)
	w := httptest.NewRecorder()
	h.handleUploadAvatar(w, req)

	// Should not be a 200.
	if w.Code == http.StatusOK {
		t.Fatalf("expected non-200 status when storage fails, got %d", w.Code)
	}

	// Verify the user's avatar_url is still nil in the repo (DB not updated).
	svc := h.service
	updatedUser, err := svc.GetByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updatedUser.AvatarURL != nil {
		t.Errorf("avatar_url should still be nil after storage error, got %q", *updatedUser.AvatarURL)
	}
}

// TestDeleteAvatar_DeletesAllExtensionsAndClearsURL verifies that
// DELETE /me/avatar calls storage.Delete for all three extensions and then
// calls UpdateAvatarURL with nil (clearing the URL in the DB).
func TestDeleteAvatar_DeletesAllExtensionsAndClearsURL(t *testing.T) {
	// Seed a user with an existing avatar URL.
	userID := uuid.MustParse("01960000-0000-7000-8000-000000000001")
	existingURL := "/files/avatars/" + userID.String() + ".png"
	user := &domain.User{
		ID:        userID,
		Email:     "test@example.com",
		Role:      "member",
		AvatarURL: &existingURL,
	}

	repo := newFakeUserRepo()
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svc := app.NewService(repo)
	fs := &fakeStorage{}
	h := &Handler{service: svc, storage: fs}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/avatar", nil)
	ctx := authmw.WithClaims(req.Context(), &authmw.Claims{
		UserID: userID,
		Email:  "test@example.com",
		Role:   "member",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.handleDeleteAvatar(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// All three extensions must be deleted.
	wantDeletes := []string{
		"avatars/" + userID.String() + ".jpg",
		"avatars/" + userID.String() + ".png",
		"avatars/" + userID.String() + ".webp",
	}
	deleteSet := make(map[string]bool)
	for _, k := range fs.deleteCalls {
		deleteSet[k] = true
	}
	for _, want := range wantDeletes {
		if !deleteSet[want] {
			t.Errorf("Delete not called for key %q; got: %v", want, fs.deleteCalls)
		}
	}

	// avatar_url must be nil after delete.
	updated, err := svc.GetByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.AvatarURL != nil {
		t.Errorf("avatar_url should be nil after delete, got %q", *updated.AvatarURL)
	}
}

// buildImagePayload returns a byte slice whose first bytes match the magic
// signature for contentType (so http.DetectContentType identifies it correctly)
// padded to the requested length with deterministic filler.
func buildImagePayload(contentType string, size int) []byte {
	ct := contentType
	if ct == "image/jpg" {
		ct = "image/jpeg"
	}
	magic, ok := imageMagic[ct]
	if !ok {
		return bytes.Repeat([]byte("a"), size)
	}
	if size <= len(magic) {
		return magic[:size]
	}
	buf := make([]byte, size)
	copy(buf, magic)
	for i := len(magic); i < size; i++ {
		buf[i] = 'a'
	}
	return buf
}

// TestUploadAvatar_RejectsSpoofedContentType verifies that a file declared as
// PNG but whose bytes are actually plain text is rejected (defence against MIME
// spoofing). storage.Put must NOT be called.
func TestUploadAvatar_RejectsSpoofedContentType(t *testing.T) {
	fs := &fakeStorage{}
	h, _, userID := newTestHandler(t, fs)

	// Build a multipart body where Content-Type says image/png but the body is
	// raw text without any image magic.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mh := make(textproto.MIMEHeader)
	mh.Set("Content-Disposition", `form-data; name="avatar"; filename="fake.png"`)
	mh.Set("Content-Type", "image/png")
	part, err := mw.CreatePart(mh)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := part.Write([]byte("<?php system($_GET['cmd']); ?>")); err != nil {
		t.Fatalf("write part: %v", err)
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/avatar", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	ctx := authmw.WithClaims(req.Context(), &authmw.Claims{UserID: userID, Email: "test@example.com", Role: "member"})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.handleUploadAvatar(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d — body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	if len(fs.putCalls) != 0 {
		t.Errorf("Put should not have been called for spoofed content-type, got %d calls", len(fs.putCalls))
	}
}
