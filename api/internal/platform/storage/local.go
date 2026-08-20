package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Local is an ObjectStorage implementation that writes files to the local
// filesystem. It is intended for development and CI environments where an
// S3-compatible service is not available.
//
// Presigned URLs are approximated with a short-lived HMAC-signed token
// (key + expiry), validated by ValidateToken on the local-upload/local-
// download endpoint. This mirrors S3's presigned-URL authorization model:
// a link alone is never sufficient proof of access (spec §6.1).
type Local struct {
	baseDir       string
	publicBaseURL string
	signingSecret string
}

// Option configures optional Local settings.
type Option func(*Local)

// WithSigningSecret sets the HMAC secret used to sign and validate presign
// tokens. Required for PresignPut/PresignGet/ValidateToken to work.
func WithSigningSecret(secret string) Option {
	return func(l *Local) {
		l.signingSecret = secret
	}
}

// NewLocal constructs a Local storage backend.
// baseDir is the root directory where files are written (e.g. "/var/www/files/avatars").
// publicBaseURL is prepended to keys to form download URLs (e.g. "/files/avatars").
func NewLocal(baseDir, publicBaseURL string, opts ...Option) *Local {
	l := &Local{
		baseDir:       baseDir,
		publicBaseURL: publicBaseURL,
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// Dir reports the local filesystem root files are stored under, so a
// consumer needing direct file access (e.g. the homeworks local-storage
// fallback download endpoint, ADR-0003) can serve bytes without a dedicated
// Get method on the ObjectStorage port.
func (l *Local) Dir() string {
	return l.baseDir
}

// Put writes obj.Body to disk at baseDir/obj.Key, creating any intermediate
// directories, and returns the public URL.
func (l *Local) Put(_ context.Context, obj Object) (string, error) {
	if err := l.validateKey(obj.Key); err != nil {
		return "", err
	}

	dest := filepath.Join(l.baseDir, filepath.FromSlash(obj.Key))

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("local storage mkdir %q: %w", filepath.Dir(dest), err)
	}

	f, err := os.Create(dest)
	if err != nil {
		return "", fmt.Errorf("local storage create %q: %w", dest, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, obj.Body); err != nil {
		return "", fmt.Errorf("local storage write %q: %w", dest, err)
	}

	return l.publicBaseURL + "/" + obj.Key, nil
}

// Delete removes the file at baseDir/key.
// If the file does not exist the call succeeds (idempotent).
func (l *Local) Delete(_ context.Context, key string) error {
	if err := l.validateKey(key); err != nil {
		return err
	}

	dest := filepath.Join(l.baseDir, filepath.FromSlash(key))

	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("local storage remove %q: %w", dest, err)
	}

	return nil
}

// PresignPut returns a local upload URL carrying a short-lived signed token
// for key. contentType and size are accepted for interface parity with the
// S3 backend but are not encoded in the token — the local backend trusts the
// caller to send matching values to the upload endpoint.
func (l *Local) PresignPut(_ context.Context, key, _ string, _ int64, ttl time.Duration) (string, error) {
	return l.presign(key, ttl)
}

// PresignGet returns a local download URL carrying a short-lived signed
// token for key.
func (l *Local) PresignGet(_ context.Context, key string, ttl time.Duration) (string, error) {
	return l.presign(key, ttl)
}

// presign builds a URL of the form "<publicBaseURL>/<key>?token=<token>"
// where token authorizes access to key until now+ttl.
func (l *Local) presign(key string, ttl time.Duration) (string, error) {
	expiry := time.Now().Add(ttl)
	token, err := l.signToken(key, expiry)
	if err != nil {
		return "", err
	}

	q := url.Values{}
	q.Set("token", token)

	return fmt.Sprintf("%s/%s?%s", l.publicBaseURL, key, q.Encode()), nil
}

// signToken builds a token of the form "<expiryUnix>.<hexHMAC>" signing
// key+expiry with the configured secret.
func (l *Local) signToken(key string, expiry time.Time) (string, error) {
	if l.signingSecret == "" {
		return "", errors.New("local storage: signing secret not configured")
	}

	expUnix := strconv.FormatInt(expiry.Unix(), 10)
	sig := l.sign(key, expUnix)

	return expUnix + "." + sig, nil
}

// sign computes the hex-encoded HMAC-SHA256 over key+expiry.
func (l *Local) sign(key, expUnix string) string {
	mac := hmac.New(sha256.New, []byte(l.signingSecret))
	mac.Write([]byte(key))
	mac.Write([]byte("|"))
	mac.Write([]byte(expUnix))

	return hex.EncodeToString(mac.Sum(nil))
}

// ValidateToken checks that token was issued for key and has not expired as
// of now. A token issued for a different key, an expired token, or a
// malformed token are all rejected — a link alone is never proof of access
// (spec §6.1).
func (l *Local) ValidateToken(key, token string, now time.Time) error {
	if l.signingSecret == "" {
		return errors.New("local storage: signing secret not configured")
	}

	expUnixStr, sig, ok := strings.Cut(token, ".")
	if !ok {
		return errors.New("local storage: malformed token")
	}

	expUnix, err := strconv.ParseInt(expUnixStr, 10, 64)
	if err != nil {
		return fmt.Errorf("local storage: malformed token expiry: %w", err)
	}

	wantSig := l.sign(key, expUnixStr)
	if subtle.ConstantTimeCompare([]byte(sig), []byte(wantSig)) != 1 {
		return errors.New("local storage: token signature does not match key")
	}

	if now.After(time.Unix(expUnix, 0)) {
		return errors.New("local storage: token expired")
	}

	return nil
}

// Head reports whether the file at baseDir/key exists and, if so, its size
// in bytes. A missing object is not an error.
func (l *Local) Head(_ context.Context, key string) (int64, bool, error) {
	if err := l.validateKey(key); err != nil {
		return 0, false, err
	}

	dest := filepath.Join(l.baseDir, filepath.FromSlash(key))

	info, err := os.Stat(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("local storage stat %q: %w", dest, err)
	}

	return info.Size(), true, nil
}

// validateKey rejects keys that could escape the baseDir via path traversal.
func (l *Local) validateKey(key string) error {
	if strings.HasPrefix(key, "/") {
		return fmt.Errorf("invalid key %q: must not be absolute", key)
	}

	// filepath.Clean collapses ".." sequences so we can detect escape attempts.
	clean := filepath.Clean(filepath.FromSlash(key))

	// After cleaning, a traversal attempt like "../etc/passwd" starts with "..".
	if strings.HasPrefix(clean, "..") {
		return fmt.Errorf("invalid key %q: path traversal detected", key)
	}

	// Double-check: the resolved path must be under baseDir.
	full := filepath.Join(l.baseDir, clean)
	if !strings.HasPrefix(full, filepath.Clean(l.baseDir)+string(os.PathSeparator)) &&
		full != filepath.Clean(l.baseDir) {
		return fmt.Errorf("invalid key %q: resolves outside base directory", key)
	}

	return nil
}
