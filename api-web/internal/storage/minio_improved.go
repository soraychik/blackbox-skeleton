package storage

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOImprovedClient struct {
	Client *minio.Client
	Bucket string
}

func NewMinIOImprovedClient() (*MinIOImprovedClient, error) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "minio:9000"
	}

	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "minioadmin"
	}

	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "minioadmin123"
	}

	useSSL := os.Getenv("MINIO_USE_SSL") == "true"

	bucketName := os.Getenv("MINIO_BUCKET_NAME")
	if bucketName == "" {
		bucketName = "blackbox-configs"
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
			Region: "us-east-1",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("Created bucket %s", bucketName)
	}

	return &MinIOImprovedClient{
		Client: client,
		Bucket: bucketName,
	}, nil
}

func (m *MinIOImprovedClient) GenerateStoragePath(deviceID int, hash string, storageType string) string {
	now := time.Now()
	yyyy := now.Format("2006")
	mm := now.Format("01")
	dd := now.Format("02")

	if storageType == "base" {
		return fmt.Sprintf("configs/%d/%s/%s/%s/%s.txt", deviceID, yyyy, mm, dd, hash)
	}
	return fmt.Sprintf("diffs/%d/%s/%s/%s/%s.patch", deviceID, yyyy, mm, dd, hash)
}

func (m *MinIOImprovedClient) UploadCompressed(ctx context.Context, objectName string, content []byte, contentType string) (int64, int64, error) {
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)

	if _, err := gz.Write(content); err != nil {
		return 0, 0, fmt.Errorf("failed to compress content: %w", err)
	}

	if err := gz.Close(); err != nil {
		return 0, 0, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	originalSize := int64(len(content))
	compressedSize := int64(compressed.Len())

	_, err := m.Client.PutObject(ctx, m.Bucket, objectName, &compressed, compressedSize, minio.PutObjectOptions{
		ContentType:     contentType,
		ContentEncoding: "gzip",
	})

	if err != nil {
		return 0, 0, fmt.Errorf("failed to upload compressed object %s: %w", objectName, err)
	}

	log.Printf("Successfully uploaded compressed object %s: %d -> %d bytes (%.1f%% reduction)",
		objectName, originalSize, compressedSize, float64(originalSize-compressedSize)/float64(originalSize)*100)

	return originalSize, compressedSize, nil
}

func (m *MinIOImprovedClient) DownloadDecompressed(ctx context.Context, objectName string) ([]byte, error) {
	object, err := m.Client.GetObject(ctx, m.Bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s: %w", objectName, err)
	}
	defer object.Close()

	gz, err := gzip.NewReader(object)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader for %s: %w", objectName, err)
	}
	defer gz.Close()

	content, err := io.ReadAll(gz)
	if err != nil {
		return nil, fmt.Errorf("failed to read decompressed content for %s: %w", objectName, err)
	}

	return content, nil
}

func (m *MinIOImprovedClient) UploadFullConfig(ctx context.Context, deviceID int, content []byte) (string, int64, int64, error) {
	hash := ""
	if len(content) > 0 {
		hash = fmt.Sprintf("%x", sha256.Sum256(content))
	}

	objectName := m.GenerateStoragePath(deviceID, hash, "base")

	originalSize, compressedSize, err := m.UploadCompressed(ctx, objectName, content, "text/plain")
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to upload full config: %w", err)
	}

	return objectName, originalSize, compressedSize, nil
}

func (m *MinIOImprovedClient) UploadDiff(ctx context.Context, deviceID int, diffContent []byte) (string, int64, int64, error) {
	hash := fmt.Sprintf("%x", sha256.Sum256(diffContent))

	objectName := m.GenerateStoragePath(deviceID, hash, "diff")

	originalSize, compressedSize, err := m.UploadCompressed(ctx, objectName, diffContent, "text/plain")
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to upload diff: %w", err)
	}

	return objectName, originalSize, compressedSize, nil
}

func (m *MinIOImprovedClient) DownloadConfig(ctx context.Context, objectName string) ([]byte, error) {
	return m.DownloadDecompressed(ctx, objectName)
}

func (m *MinIOImprovedClient) ListDeviceObjects(ctx context.Context, deviceID int) ([]string, error) {
	prefix := fmt.Sprintf("%d/", deviceID)

	var objects []string
	for object := range m.Client.ListObjects(ctx, m.Bucket, minio.ListObjectsOptions{
		Prefix: prefix,
	}) {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}
		objects = append(objects, object.Key)
	}

	return objects, nil
}

func (m *MinIOImprovedClient) DeleteObject(ctx context.Context, objectName string) error {
	err := m.Client.RemoveObject(ctx, m.Bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object %s: %w", objectName, err)
	}

	log.Printf("Successfully deleted object %s", objectName)
	return nil
}

func (m *MinIOImprovedClient) GetObjectInfo(ctx context.Context, objectName string) (minio.ObjectInfo, error) {
	return m.Client.StatObject(ctx, m.Bucket, objectName, minio.StatObjectOptions{})
}

func (m *MinIOImprovedClient) GeneratePresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	url, err := m.Client.PresignedGetObject(ctx, m.Bucket, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for %s: %w", objectName, err)
	}

	return url.String(), nil
}

func (m *MinIOImprovedClient) GetStorageStats(ctx context.Context, deviceID int) (map[string]int64, error) {
	prefix := fmt.Sprintf("%d/", deviceID)

	objects, err := m.ListObjects(prefix)
	if err != nil {
		return nil, err
	}

	stats := map[string]int64{
		"configs": 0,
		"diffs":   0,
		"total":   0,
	}

	for _, object := range objects {
		info, err := m.GetObjectInfo(ctx, object)
		if err != nil {
			log.Printf("Warning: failed to get object info for %s: %v", object, err)
			continue
		}

		stats["total"] += info.Size

		if strings.HasPrefix(object, "configs/") {
			stats["configs"] += info.Size
		} else if strings.HasPrefix(object, "diffs/") {
			stats["diffs"] += info.Size
		}
	}

	return stats, nil
}

func (m *MinIOImprovedClient) ListObjects(prefix string) ([]string, error) {
	ctx := context.Background()
	var objects []string
	for object := range m.Client.ListObjects(ctx, m.Bucket, minio.ListObjectsOptions{
		Prefix: prefix,
	}) {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}
		objects = append(objects, object.Key)
	}
	return objects, nil
}
