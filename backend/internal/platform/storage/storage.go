package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var ErrNotFound = errors.New("storage object not found")

type Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

type Object struct {
	Key          string
	LastModified time.Time
}

type Client struct {
	minio  *minio.Client
	bucket string
}

func New(cfg Config) (*Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}
	return &Client{minio: client, bucket: cfg.Bucket}, nil
}

func (c *Client) PresignPut(ctx context.Context, key, contentType string, expiry time.Duration) (string, error) {
	signed, err := c.minio.PresignHeader(ctx, http.MethodPut, c.bucket, key, expiry, nil, http.Header{"Content-Type": {contentType}})
	if err != nil {
		return "", fmt.Errorf("presign put %s: %w", key, err)
	}
	return signed.String(), nil
}

func (c *Client) Size(ctx context.Context, key string) (int64, error) {
	info, err := c.minio.StatObject(ctx, c.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return 0, wrap("stat", key, err)
	}
	return info.Size, nil
}

func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	object, err := c.minio.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, wrap("get", key, err)
	}
	defer object.Close()
	data, err := io.ReadAll(object)
	if err != nil {
		return nil, wrap("read", key, err)
	}
	return data, nil
}

func (c *Client) Put(ctx context.Context, key string, data []byte, contentType, cacheControl string) error {
	_, err := c.minio.PutObject(ctx, c.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType, CacheControl: cacheControl,
	})
	return wrap("put", key, err)
}

func (c *Client) Remove(ctx context.Context, key string) error {
	err := wrap("remove", key, c.minio.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{}))
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (c *Client) List(ctx context.Context, prefix string) ([]Object, error) {
	var objects []Object
	for info := range c.minio.ListObjects(ctx, c.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if info.Err != nil {
			return nil, fmt.Errorf("list %s: %w", prefix, info.Err)
		}
		objects = append(objects, Object{Key: info.Key, LastModified: info.LastModified})
	}
	return objects, nil
}

func wrap(op, key string, err error) error {
	if err == nil {
		return nil
	}
	if code := minio.ToErrorResponse(err).Code; code == "NoSuchKey" || code == "NotFound" {
		return fmt.Errorf("%s %s: %w", op, key, ErrNotFound)
	}
	return fmt.Errorf("%s %s: %w", op, key, err)
}
