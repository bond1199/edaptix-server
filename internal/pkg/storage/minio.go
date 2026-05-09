package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/edaptix/server/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOProvider struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

func NewMinIOProvider(cfg config.MinIOConfig) (*MinIOProvider, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	p := &MinIOProvider{
		client:    client,
		bucket:    cfg.Bucket,
		publicURL: cfg.PublicURL,
	}

	if err := p.EnsureBucketExists(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return p, nil
}

func (p *MinIOProvider) Upload(ctx context.Context, objectName string, reader io.Reader, objectSize int64, contentType string) (string, error) {
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}

	info, err := p.client.PutObject(ctx, p.bucket, objectName, reader, objectSize, opts)
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %w", err)
	}

	fileURL := fmt.Sprintf("%s/%s/%s", p.publicURL, p.bucket, info.Key)
	return fileURL, nil
}

func (p *MinIOProvider) GetPresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	presignedURL, err := p.client.PresignedGetObject(ctx, p.bucket, objectName, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL: %w", err)
	}

	return presignedURL.String(), nil
}

func (p *MinIOProvider) Delete(ctx context.Context, objectName string) error {
	err := p.client.RemoveObject(ctx, p.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

func (p *MinIOProvider) EnsureBucketExists(ctx context.Context) error {
	exists, err := p.client.BucketExists(ctx, p.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		if err := p.client.MakeBucket(ctx, p.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}
