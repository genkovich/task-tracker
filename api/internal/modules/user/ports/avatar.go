package ports

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	authmw "github.com/genkovich/task-tracker/api/internal/platform/authmw"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
	"github.com/genkovich/task-tracker/api/internal/platform/storage"
)

const maxAvatarBytes = 2 * 1024 * 1024 // 2 MiB

var avatarExtensions = []string{"jpg", "png", "webp"}

func extForContentType(ct string) string {
	switch ct {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	default:
		return ""
	}
}

func avatarKey(userID, ext string) string {
	return fmt.Sprintf("avatars/%s.%s", userID, ext)
}

// sniffContentType reads the first 512 bytes to verify the declared MIME type
// matches the actual file bytes. Returns the remaining body reader (with the
// sniffed bytes re-prepended) so callers can stream the full file downstream.
func sniffContentType(file io.Reader, declared string) (body io.Reader, ok bool) {
	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	head = head[:n]
	detected := http.DetectContentType(head)
	if detected != declared {
		// http.DetectContentType returns "image/jpeg" for jpg; map our alias.
		if declared != "image/jpg" || detected != "image/jpeg" {
			return nil, false
		}
	}
	return io.MultiReader(bytes.NewReader(head), file), true
}

func (h *Handler) handleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	claims, ok := authmw.AuthClaims(r.Context())
	if !ok {
		httputil.WriteValidationError(w, "auth.missing_token", "authorization token is required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarBytes+1024)
	if err := r.ParseMultipartForm(maxAvatarBytes + 1024); err != nil {
		httputil.WriteValidationError(w, "validation.avatar_too_large", "avatar must be 2MB or smaller")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		httputil.WriteValidationError(w, "validation.avatar_required", "avatar file is required (field \"avatar\")")
		return
	}
	defer file.Close()

	if header.Size > maxAvatarBytes {
		httputil.WriteValidationError(w, "validation.avatar_too_large", "avatar must be 2MB or smaller")
		return
	}

	contentType := header.Header.Get("Content-Type")
	ext := extForContentType(contentType)
	if ext == "" {
		httputil.WriteValidationError(w, "validation.avatar_type", "avatar must be a jpg, png, or webp image")
		return
	}

	body, ok := sniffContentType(file, contentType)
	if !ok {
		httputil.WriteValidationError(w, "validation.avatar_type", "file content does not match declared type")
		return
	}

	ctx := r.Context()
	userID := claims.UserID.String()
	key := avatarKey(userID, ext)

	publicURL, err := h.storage.Put(ctx, storage.Object{
		Key:         key,
		ContentType: contentType,
		Body:        body,
		Size:        header.Size,
	})
	if err != nil {
		slog.ErrorContext(ctx, "avatar storage put failed", "user_id", userID, "key", key, "error", err)
		httputil.WriteError(w, errors.New("avatar storage put failed"))
		return
	}

	// Best-effort cleanup of old variants. Delete is idempotent at the storage
	// layer, so missing-object errors are silently dropped. Other failures are
	// logged but do not block the successful upload.
	for _, e := range avatarExtensions {
		if e == ext {
			continue
		}
		if err := h.storage.Delete(ctx, avatarKey(userID, e)); err != nil {
			slog.WarnContext(ctx, "avatar cleanup delete failed", "user_id", userID, "ext", e, "error", err)
		}
	}

	urlWithBuster := fmt.Sprintf("%s?v=%d", publicURL, time.Now().Unix())
	user, err := h.service.UpdateAvatarURL(ctx, claims.UserID, &urlWithBuster)
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}

	httputil.WriteJSON(w, toUserResponse(user), http.StatusOK)
}

func (h *Handler) handleDeleteAvatar(w http.ResponseWriter, r *http.Request) {
	claims, ok := authmw.AuthClaims(r.Context())
	if !ok {
		httputil.WriteValidationError(w, "auth.missing_token", "authorization token is required")
		return
	}

	ctx := r.Context()
	userID := claims.UserID.String()

	for _, e := range avatarExtensions {
		if err := h.storage.Delete(ctx, avatarKey(userID, e)); err != nil {
			slog.WarnContext(ctx, "avatar delete failed", "user_id", userID, "ext", e, "error", err)
		}
	}

	user, err := h.service.UpdateAvatarURL(ctx, claims.UserID, nil)
	if err != nil {
		httputil.WriteError(w, mapError(err))
		return
	}
	httputil.WriteJSON(w, toUserResponse(user), http.StatusOK)
}
