package storage

import (
	"context"
	"io"
	"time"
)

type StorageClient interface {
	Upload(ctx context.Context, filename string, content io.Reader, size int64, contentType string) (string, error)
	Delete(ctx context.Context, key string) error
	Download(ctx context.Context, key string) ([]byte, error)
	GetPresignedURL(ctx context.Context, key string, expires time.Duration) (string, error)
}

type NoOpStorage struct{}

//#region Upload
func (n *NoOpStorage) Upload(ctx context.Context, f string, r io.Reader, s int64, ct string) (string, error) {
	return f, nil
}
//#region Delete
func (n *NoOpStorage) Delete(ctx context.Context, k string) error             { return nil }
//#region Download
func (n *NoOpStorage) Download(ctx context.Context, k string) ([]byte, error) { return nil, nil }
//#region GetPresignedURL
func (n *NoOpStorage) GetPresignedURL(ctx context.Context, k string, e time.Duration) (string, error) {
	return "http://localhost/noop.pdf", nil
}
