package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	RawBucket string
	HLSBucket string
}

type Store struct {
	client    *minio.Client
	rawBucket string
	hlsBucket string
}

func New(ctx context.Context, cfg Config) (*Store, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: init failed: %w", err)
	}

	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for _, bucket := range []string{cfg.RawBucket, cfg.HLSBucket} {
		if err := ensureBucket(checkCtx, client, bucket); err != nil {
			return nil, err
		}
	}

	return &Store{client: client, rawBucket: cfg.RawBucket, hlsBucket: cfg.HLSBucket}, nil
}

// PresignedPutRaw returns a presigned PUT URL for uploading a raw video to the
// raw bucket under objectKey, valid for expiry.
func (s *Store) PresignedPutRaw(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedPutObject(ctx, s.rawBucket, objectKey, expiry)
	if err != nil {
		return "", fmt.Errorf("minio: presign put: %w", err)
	}
	return u.String(), nil
}

// RawBucket exposes the raw upload bucket name.
func (s *Store) RawBucket() string { return s.rawBucket }

// HLSBucket exposes the HLS output bucket name.
func (s *Store) HLSBucket() string { return s.hlsBucket }

// DownloadRaw fetches objectKey from the raw bucket to local path dst.
func (s *Store) DownloadRaw(ctx context.Context, objectKey, dst string) error {
	if err := s.client.FGetObject(ctx, s.rawBucket, objectKey, dst, minio.GetObjectOptions{}); err != nil {
		return fmt.Errorf("storage: download %q: %w", objectKey, err)
	}
	return nil
}

// UploadHLSDir uploads every file under localDir to the HLS bucket, keyed under
// prefix using paths relative to localDir. Content types are inferred from the
// file extension for the common HLS artifacts.
func (s *Store) UploadHLSDir(ctx context.Context, localDir, prefix string) error {
	return filepath.WalkDir(localDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(filepath.Join(prefix, rel))
		_, err = s.client.FPutObject(ctx, s.hlsBucket, key, path, minio.PutObjectOptions{
			ContentType: contentTypeFor(path),
		})
		if err != nil {
			return fmt.Errorf("storage: upload %q: %w", key, err)
		}
		return nil
	})
}

func contentTypeFor(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".m3u8":
		return "application/vnd.apple.mpegurl"
	case ".ts":
		return "video/mp2t"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".vtt":
		return "text/vtt"
	case ".key":
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}

func ensureBucket(ctx context.Context, client *minio.Client, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("minio: check bucket %q: %w", bucket, err)
	}
	if exists {
		return nil
	}
	if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("minio: create bucket %q: %w", bucket, err)
	}
	return nil
}
