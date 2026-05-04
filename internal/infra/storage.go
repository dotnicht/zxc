package infra

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type StorageClient struct {
	mc     *minio.Client
	prefix string
}

func StorageClientFromConnectionString(dsn string) (*StorageClient, string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, "", fmt.Errorf("invalid storage DSN: %w", err)
	}
	var accessKey, secretKey string
	if u.User != nil {
		accessKey = u.User.Username()
		secretKey, _ = u.User.Password()
	}
	parts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)
	bucket := parts[0]
	var prefix string
	if len(parts) > 1 && parts[1] != "" {
		prefix = parts[1] + "/"
	}
	mc, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: u.Scheme == "https",
	})
	if err != nil {
		return nil, "", fmt.Errorf("storage client: %w", err)
	}
	return &StorageClient{mc: mc, prefix: prefix}, bucket, nil
}

func (c *StorageClient) CreateFolder(ctx context.Context, bucket, folder string) error {
	exists, err := c.mc.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := c.mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
	}
	key := strings.TrimRight(folder, "/") + "/"
	_, err = c.mc.PutObject(ctx, bucket, key, bytes.NewReader([]byte{}), 0, minio.PutObjectOptions{})
	return err
}

func (c *StorageClient) Upload(ctx context.Context, bucket, path string, r io.Reader, size int64, contentType string) error {
	key := c.prefix + path
	_, err := c.mc.PutObject(ctx, bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("upload %s/%s: %w", bucket, key, err)
	}
	return nil
}

func (c *StorageClient) Download(ctx context.Context, bucket, path string) (io.ReadCloser, error) {
	key := c.prefix + path
	obj, err := c.mc.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("download %s/%s: %w", bucket, key, err)
	}
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		return nil, fmt.Errorf("stat %s/%s: %w", bucket, key, err)
	}
	return obj, nil
}

func (c *StorageClient) Delete(ctx context.Context, bucket, path string) error {
	return c.mc.RemoveObject(ctx, bucket, c.prefix+path, minio.RemoveObjectOptions{})
}
