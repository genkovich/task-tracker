package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// S3Config holds all parameters required to construct an S3-compatible client.
type S3Config struct {
	// Bucket is the target S3 bucket name.
	Bucket string
	// Region is the AWS or Contabo region identifier (e.g. "eu2").
	Region string
	// Endpoint is the full base URL of the S3-compatible service
	// (e.g. "https://eu2.contabostorage.com").
	Endpoint string
	// PublicBaseURL is prepended to the key to form the public download URL
	// (e.g. "https://eu2.contabostorage.com/task-tracker-avatars").
	PublicBaseURL string
	// AccessKeyID is the S3 access key identifier.
	AccessKeyID string
	// SecretKey is the S3 secret access key.
	SecretKey string
}

// S3 is an ObjectStorage implementation backed by an S3-compatible service.
type S3 struct {
	client  *s3.Client
	presign *s3.PresignClient
	cfg     S3Config
}

// NewS3 constructs an S3 client using static credentials and path-style
// addressing required by Contabo Object Storage.
func NewS3(cfg S3Config) (*S3, error) {
	creds := credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretKey, "")

	awsCfg := aws.Config{
		Region:      cfg.Region,
		Credentials: creds,
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true
	})

	return &S3{client: client, presign: s3.NewPresignClient(client), cfg: cfg}, nil
}

// Put uploads obj to S3 and returns the public URL constructed from
// PublicBaseURL + "/" + obj.Key.
func (s *S3) Put(ctx context.Context, obj Object) (string, error) {
	var contentLength *int64
	if obj.Size > 0 {
		contentLength = aws.Int64(obj.Size)
	}

	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.cfg.Bucket),
		Key:           aws.String(obj.Key),
		Body:          obj.Body,
		ContentType:   aws.String(obj.ContentType),
		ContentLength: contentLength,
	}

	if _, err := s.client.PutObject(ctx, input); err != nil {
		return "", fmt.Errorf("s3 put %q: %w", obj.Key, err)
	}

	return fmt.Sprintf("%s/%s", s.cfg.PublicBaseURL, obj.Key), nil
}

// Delete removes the object identified by key from S3.
// If the object does not exist the call succeeds (idempotent).
func (s *S3) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	}

	if _, err := s.client.DeleteObject(ctx, input); err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("s3 delete %q: %w", key, err)
	}

	return nil
}

// PresignPut returns a URL that lets a client upload directly to S3, valid
// for at most ttl. Presigning is a local computation — no network round-trip.
func (s *S3) PresignPut(ctx context.Context, key, contentType string, size int64, ttl time.Duration) (string, error) {
	var contentLength *int64
	if size > 0 {
		contentLength = aws.Int64(size)
	}

	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.cfg.Bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(contentType),
		ContentLength: contentLength,
	}

	req, err := s.presign.PresignPutObject(ctx, input, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("s3 presign put %q: %w", key, err)
	}

	return req.URL, nil
}

// PresignGet returns a URL that lets a client download directly from S3,
// valid for at most ttl.
func (s *S3) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	}

	req, err := s.presign.PresignGetObject(ctx, input, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("s3 presign get %q: %w", key, err)
	}

	return req.URL, nil
}

// Head reports whether the object identified by key exists in S3 and, if
// so, its size in bytes. A missing object is not an error.
func (s *S3) Head(ctx context.Context, key string) (int64, bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	}

	out, err := s.client.HeadObject(ctx, input)
	if err != nil {
		if isNotFound(err) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("s3 head %q: %w", key, err)
	}

	size := int64(0)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}

	return size, true, nil
}

// isNotFound returns true when the error represents a missing S3 object.
func isNotFound(err error) bool {
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}

	// Contabo and some S3-compatible services return a generic API error with
	// code "NoSuchKey" or "NotFound" instead of the typed error.
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == "NoSuchKey" || code == "NotFound"
	}

	return false
}
