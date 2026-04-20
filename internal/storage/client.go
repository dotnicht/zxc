package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

type Client struct {
	mc           *minio.Client
	cfg          Config
	objectPrefix string
}

type BucketInfo struct {
	Name      string
	CreatedAt time.Time
}

type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	IsDir        bool
}

func NewClient(cfg Config) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}
	return &Client{mc: mc, cfg: cfg}, nil
}

func (c *Client) CreateBucket(ctx context.Context, name string) error {
	exists, err := c.mc.BucketExists(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if exists {
		return nil
	}
	if err := c.mc.MakeBucket(ctx, name, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}
	return nil
}

func (c *Client) CreateFolder(ctx context.Context, bucket, folder string) error {
	exists, err := c.mc.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		if err := c.mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	key := strings.TrimRight(folder, "/") + "/"
	_, err = c.mc.PutObject(ctx, bucket, key, bytes.NewReader([]byte{}), 0, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to create folder: %w", err)
	}
	return nil
}

func (c *Client) BucketConnectionString(bucketName string) string {
	scheme := "http"
	if c.cfg.UseSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%s@%s/%s", scheme, c.cfg.AccessKey, c.cfg.SecretKey, c.cfg.Endpoint, bucketName)
}

func ParseConnectionString(connStr string) (Config, string, string, error) {
	u, err := url.Parse(connStr)
	if err != nil {
		return Config{}, "", "", fmt.Errorf("invalid storage connection string: %w", err)
	}
	var accessKey, secretKey string
	if u.User != nil {
		accessKey = u.User.Username()
		secretKey, _ = u.User.Password()
	}
	rawPath := strings.TrimPrefix(u.Path, "/")
	parts := strings.SplitN(rawPath, "/", 2)
	bucketName := parts[0]
	var objectPrefix string
	if len(parts) > 1 && parts[1] != "" {
		objectPrefix = parts[1] + "/"
	}
	cfg := Config{
		Endpoint:  u.Host,
		AccessKey: accessKey,
		SecretKey: secretKey,
		UseSSL:    u.Scheme == "https",
	}
	return cfg, bucketName, objectPrefix, nil
}

func ClientFromConnectionString(connStr string) (*Client, string, error) {
	cfg, bucketName, objectPrefix, err := ParseConnectionString(connStr)
	if err != nil {
		return nil, "", err
	}
	client, err := NewClient(cfg)
	if err != nil {
		return nil, "", err
	}
	client.objectPrefix = objectPrefix
	return client, bucketName, nil
}

func (c *Client) ListBuckets(ctx context.Context) ([]BucketInfo, error) {
	buckets, err := c.mc.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list buckets: %w", err)
	}

	out := make([]BucketInfo, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, BucketInfo{
			Name:      b.Name,
			CreatedAt: b.CreationDate,
		})
	}
	return out, nil
}

func (c *Client) ListObjects(ctx context.Context, bucket, prefix string, recursive bool) ([]ObjectInfo, error) {
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: recursive,
	}

	objects := make([]ObjectInfo, 0)
	for obj := range c.mc.ListObjects(ctx, bucket, opts) {
		if obj.Err != nil {
			return nil, fmt.Errorf("list objects %s/%s: %w", bucket, prefix, obj.Err)
		}
		objects = append(objects, ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			IsDir:        strings.HasSuffix(obj.Key, "/"),
		})
	}
	return objects, nil
}

func (c *Client) Upload(ctx context.Context, bucket, objectPath string, r io.Reader, size int64, contentType string) error {
	key := c.objectPrefix + objectPath
	_, err := c.mc.PutObject(ctx, bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("put object %s/%s: %w", bucket, key, err)
	}
	return nil
}

func (c *Client) Download(ctx context.Context, bucket, objectPath string) (io.ReadCloser, error) {
	key := c.objectPrefix + objectPath
	obj, err := c.mc.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %s/%s: %w", bucket, key, err)
	}

	if _, err := obj.Stat(); err != nil {
		obj.Close()
		return nil, fmt.Errorf("stat object %s/%s: %w", bucket, key, err)
	}
	return obj, nil
}

func BucketName(tenantName string) string {
	var result strings.Builder
	for _, r := range strings.ToLower(tenantName) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		} else {
			result.WriteRune('-')
		}
	}
	name := strings.Trim(result.String(), "-")
	if len(name) < 3 {
		name = name + "---"
		name = name[:3]
	}
	if len(name) > 63 {
		name = name[:63]
	}
	return name
}
