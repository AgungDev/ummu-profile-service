package transport

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLocalAvatarFilesArePublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	directory := t.TempDir()
	avatar := filepath.Join(directory, "user-1", "avatar.png")
	if err := os.MkdirAll(filepath.Dir(avatar), 0o750); err != nil {
		t.Fatalf("create avatar directory: %v", err)
	}
	if err := os.WriteFile(avatar, []byte("avatar content"), 0o600); err != nil {
		t.Fatalf("write avatar: %v", err)
	}

	router := gin.New()
	if err := registerLocalAvatarFiles(router, directory); err != nil {
		t.Fatalf("register avatar route: %v", err)
	}
	router.GET("/api/v1/profiles/:user_id", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/profiles/avatars/user-1/avatar.png",
		nil,
	))
	if response.Code != http.StatusOK || response.Body.String() != "avatar content" {
		t.Fatalf("avatar response = %d %q", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=86400" {
		t.Fatalf("Cache-Control = %q", got)
	}
}

func TestLocalAvatarFilesDoNotServePathTraversal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	parent := t.TempDir()
	directory := filepath.Join(parent, "avatars")
	if err := os.Mkdir(directory, 0o750); err != nil {
		t.Fatalf("create avatar directory: %v", err)
	}
	outside := filepath.Join(parent, "outside.jpg")
	if err := os.WriteFile(outside, []byte("private"), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	t.Cleanup(func() { os.Remove(outside) })

	router := gin.New()
	if err := registerLocalAvatarFiles(router, directory); err != nil {
		t.Fatalf("register avatar route: %v", err)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/profiles/avatars/../outside.jpg",
		nil,
	))
	if response.Code == http.StatusOK && response.Body.String() == "private" {
		t.Fatal("path traversal exposed a file outside the avatar directory")
	}
}
