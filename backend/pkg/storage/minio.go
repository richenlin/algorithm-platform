package storage

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIO struct {
	client *minio.Client
}

func New(endpoint, accessKey, secretKey string, useSSL bool) (*MinIO, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return &MinIO{client: client}, nil
}

func (m *MinIO) UploadFile(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, bucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (m *MinIO) DownloadFile(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error) {
	return m.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
}

func (m *MinIO) GetPresignedURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error) {
	u, err := m.client.PresignedGetObject(ctx, bucketName, objectName, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (m *MinIO) DeleteFile(ctx context.Context, bucketName, objectName string) error {
	return m.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
}

func (m *MinIO) ListFiles(ctx context.Context, bucketName, prefix string) ([]FileInfo, error) {
	objects := m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix: prefix,
	})

	var files []FileInfo
	for obj := range objects {
		if obj.Err != nil {
			return nil, obj.Err
		}
		files = append(files, FileInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
		})
	}

	return files, nil
}

func (m *MinIO) FileExists(ctx context.Context, bucketName, objectName string) (bool, error) {
	_, err := m.client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type FileInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
}

func (m *MinIO) CreateBucket(ctx context.Context, bucketName string) error {
	exists, err := m.client.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}
	if !exists {
		return m.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	}
	return nil
}
