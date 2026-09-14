// Package oss 封装对象存储（生产用公司 OSS，本地用 MinIO，接口一致）。
//
// 用途：
//   - bucket_kb：知识库原始文档（PDF/Word/Markdown）
//   - bucket_media：多模态文件（图片、音频、视频、日志截图）
package oss

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"bokeoncall/internal/conf"
)

// Client 对象存储客户端。
type Client struct {
	c   *minio.Client
	cfg conf.MinIOConfig
}

// NewClient 创建客户端。
func NewClient(cfg conf.MinIOConfig) (*Client, error) {
	c, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("创建对象存储客户端失败: %w", err)
	}
	return &Client{c: c, cfg: cfg}, nil
}

// EnsureBuckets 确保知识库桶与多模态桶存在（幂等）。
func (c *Client) EnsureBuckets(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	for _, bucket := range []string{c.cfg.BucketKB, c.cfg.BucketMedia} {
		if bucket == "" {
			continue
		}
		exists, err := c.c.BucketExists(ctx, bucket)
		if err != nil {
			return fmt.Errorf("检查桶 %s 失败: %w", bucket, err)
		}
		if exists {
			continue
		}
		if err := c.c.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: c.cfg.Region}); err != nil {
			return fmt.Errorf("创建桶 %s 失败: %w", bucket, err)
		}
	}
	return nil
}

// Health 健康检查。
func (c *Client) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := c.c.BucketExists(ctx, c.cfg.BucketKB)
	if err != nil {
		return fmt.Errorf("对象存储不可用: %w", err)
	}
	return nil
}

// PutBytes 上传字节内容，返回 object key。
func (c *Client) PutBytes(ctx context.Context, bucket, object string, data []byte, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := c.c.PutObject(ctx, bucket, object, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("上传对象失败: %w", err)
	}
	return nil
}

// PutStream 流式上传（大文件、视频走这里，避免整块读进内存）。
func (c *Client) PutStream(ctx context.Context, bucket, object string, reader io.Reader, size int64, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := c.c.PutObject(ctx, bucket, object, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
		PartSize:    16 << 20, // 16MB 分片，适配大文件
	})
	if err != nil {
		return fmt.Errorf("流式上传对象失败: %w", err)
	}
	return nil
}

// GetBytes 下载对象。
func (c *Client) GetBytes(ctx context.Context, bucket, object string) ([]byte, error) {
	obj, err := c.c.GetObject(ctx, bucket, object, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("下载对象失败: %w", err)
	}
	defer obj.Close()
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("读取对象失败: %w", err)
	}
	return data, nil
}

// PresignedGet 生成预签名下载地址（前端/飞书卡片直接拉文件用，别走服务端中转）。
func (c *Client) PresignedGet(ctx context.Context, bucket, object string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = time.Hour
	}
	u, err := c.c.PresignedGetObject(ctx, bucket, object, ttl, nil)
	if err != nil {
		return "", fmt.Errorf("生成预签名地址失败: %w", err)
	}
	return u.String(), nil
}
