package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/AgungDev/ummu-profile-service/internal/config"
)

func TestLocalAvatarStorePersistsAndDeletesAvatar(t *testing.T) {
	directory := t.TempDir()
	publicBase := "http://localhost:7000/api/v1/profiles/avatars"
	store, err := NewAvatarStore(config.AvatarConfig{
		LocalDir:   directory,
		PublicBase: publicBase,
	})
	if err != nil {
		t.Fatalf("create local avatar store: %v", err)
	}

	key := "user-1/avatar.png"
	content := []byte("avatar content")
	if err := store.Put(context.Background(), key, "image/png", content); err != nil {
		t.Fatalf("save avatar: %v", err)
	}
	if got, want := store.URL(key), publicBase+"/"+key; got != want {
		t.Fatalf("avatar URL = %q, want %q", got, want)
	}

	path := filepath.Join(directory, filepath.FromSlash(key))
	if got, err := os.ReadFile(path); err != nil || string(got) != string(content) {
		t.Fatalf("stored avatar = %q, error = %v", got, err)
	}
	if err := store.Delete(context.Background(), key); err != nil {
		t.Fatalf("delete avatar: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("avatar still exists or stat failed: %v", err)
	}
}

func TestLocalAvatarStoreRejectsPathTraversal(t *testing.T) {
	store, err := NewAvatarStore(config.AvatarConfig{
		LocalDir:   t.TempDir(),
		PublicBase: "http://localhost/avatars",
	})
	if err != nil {
		t.Fatalf("create local avatar store: %v", err)
	}

	if err := store.Put(context.Background(), "../outside.png", "image/png", []byte("avatar")); err == nil {
		t.Fatal("expected path traversal key to be rejected")
	}
}
