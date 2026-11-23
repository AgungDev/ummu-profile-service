package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/AgungDev/ummu-profile-service/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type AvatarStore interface {
	Put(ctx context.Context, key, contentType string, data []byte) error
	Delete(ctx context.Context, key string) error
	URL(key string) string
}

type s3AvatarStore struct {
	client     *minio.Client
	bucket     string
	publicBase string
}

func NewAvatarStore(cfg config.AvatarConfig) (AvatarStore, error) {
	if cfg.LocalDir != "" {
		directory, err := filepath.Abs(cfg.LocalDir)
		if err != nil {
			return nil, fmt.Errorf("resolve local avatar directory: %w", err)
		}
		if err := os.MkdirAll(directory, 0o750); err != nil {
			return nil, fmt.Errorf("create local avatar directory: %w", err)
		}
		return &localAvatarStore{
			directory:  directory,
			publicBase: strings.TrimRight(cfg.PublicBase, "/"),
		}, nil
	}
	if cfg.Endpoint == "" {
		return nil, nil
	}
	endpoint := strings.TrimPrefix(strings.TrimPrefix(cfg.Endpoint, "https://"), "http://")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create avatar object storage client: %w", err)
	}
	return &s3AvatarStore{client: client, bucket: cfg.Bucket, publicBase: strings.TrimRight(cfg.PublicBase, "/")}, nil
}

func (s *s3AvatarStore) Put(ctx context.Context, key, contentType string, data []byte) error {
	if _, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	}); err != nil {
		return fmt.Errorf("upload avatar object: %w", err)
	}
	return nil
}

func (s *s3AvatarStore) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete avatar object: %w", err)
	}
	return nil
}

func (s *s3AvatarStore) URL(key string) string {
	return avatarURL(s.publicBase, s.bucket, key)
}

type localAvatarStore struct {
	directory  string
	publicBase string
}

func (s *localAvatarStore) Put(_ context.Context, key, _ string, data []byte) error {
	path, err := s.filePath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create avatar directory: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".avatar-*")
	if err != nil {
		return fmt.Errorf("create temporary avatar file: %w", err)
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("write avatar file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close avatar file: %w", err)
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return fmt.Errorf("save avatar file: %w", err)
	}
	return nil
}

func (s *localAvatarStore) Delete(_ context.Context, key string) error {
	path, err := s.filePath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete avatar file: %w", err)
	}
	return nil
}

func (s *localAvatarStore) URL(key string) string {
	return avatarURL(s.publicBase, "", key)
}

func (s *localAvatarStore) filePath(key string) (string, error) {
	cleanKey := filepath.Clean(filepath.FromSlash(key))
	if key == "" || filepath.IsAbs(cleanKey) || cleanKey == "." || cleanKey == ".." ||
		strings.HasPrefix(cleanKey, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid avatar key")
	}
	path := filepath.Join(s.directory, cleanKey)
	relative, err := filepath.Rel(s.directory, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid avatar key")
	}
	return path, nil
}

func avatarURL(publicBase, bucket, key string) string {
	parts := strings.Split(key, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	if bucket != "" {
		publicBase += "/" + url.PathEscape(bucket)
	}
	return publicBase + "/" + strings.Join(parts, "/")
}
