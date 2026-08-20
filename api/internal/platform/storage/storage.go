// Package storage defines the ObjectStorage port and shared types used by
// all storage back-ends (S3-compatible, local disk).
package storage

import (
	"context"
	"io"
	"time"
)

// Object carries everything needed to store one file.
type Object struct {
	// Key is the storage path, e.g. "avatars/123e4567.png".
	Key string
	// ContentType is the MIME type, e.g. "image/png".
	ContentType string
	// Body is the file content as a stream.
	Body io.Reader
	// Size is the byte count of Body. Zero means unknown.
	Size int64
}

// ObjectStorage is the port that all storage back-ends must satisfy.
type ObjectStorage interface {
	// Put stores obj and returns the public URL that clients can fetch.
	Put(ctx context.Context, obj Object) (publicURL string, err error)
	// Delete removes the object identified by key.
	// If the object does not exist the call is a no-op (idempotent).
	Delete(ctx context.Context, key string) error
	// PresignPut returns a URL the client can use to upload the object
	// identified by key directly to the storage back-end, bypassing the API
	// process. The URL is valid for at most ttl.
	PresignPut(ctx context.Context, key, contentType string, size int64, ttl time.Duration) (url string, err error)
	// PresignGet returns a URL the client can use to download the object
	// identified by key directly from the storage back-end. The URL is valid
	// for at most ttl.
	PresignGet(ctx context.Context, key string, ttl time.Duration) (url string, err error)
	// Head reports whether the object identified by key exists and, if so,
	// its size in bytes. A missing object is not an error.
	Head(ctx context.Context, key string) (size int64, exists bool, err error)
}
