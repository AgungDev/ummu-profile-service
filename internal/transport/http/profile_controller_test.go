package http

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ummuauth "github.com/AgungDev/ummu-go-auth"
	"github.com/AgungDev/ummu-profile-service/internal/config"
	"github.com/AgungDev/ummu-profile-service/internal/domain/dto"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type staticKeyProvider struct {
	key *rsa.PublicKey
}

func (p staticKeyProvider) Get(_ context.Context, keyID string) (*rsa.PublicKey, error) {
	if keyID != "test-key" {
		return nil, errors.New("unknown key")
	}
	return p.key, nil
}

type controllerUsecaseFake struct {
	updatedUserID uuid.UUID
	publicReads   int
}

func (f *controllerUsecaseFake) Get(_ context.Context, userID uuid.UUID) (*dto.ProfileResponse, error) {
	return &dto.ProfileResponse{UserID: userID.String()}, nil
}

func (f *controllerUsecaseFake) GetPublic(_ context.Context, userID uuid.UUID) (*dto.PublicProfileResponse, error) {
	f.publicReads++
	return &dto.PublicProfileResponse{UserID: userID.String(), FullName: "Admin View"}, nil
}

func (f *controllerUsecaseFake) Update(_ context.Context, userID uuid.UUID, _ dto.UpdateProfileRequest) (*dto.ProfileResponse, error) {
	f.updatedUserID = userID
	return &dto.ProfileResponse{UserID: userID.String()}, nil
}

func (f *controllerUsecaseFake) UploadAvatar(_ context.Context, userID uuid.UUID, _ string, _ []byte) (*dto.ProfileResponse, error) {
	return &dto.ProfileResponse{UserID: userID.String()}, nil
}

func (f *controllerUsecaseFake) DeleteAvatar(_ context.Context, userID uuid.UUID) (*dto.ProfileResponse, error) {
	return &dto.ProfileResponse{UserID: userID.String()}, nil
}

func TestUpdateMeUsesVerifiedSubjectInsteadOfBodyUserID(t *testing.T) {
	authenticator, privateKey := newTestAuthenticator(t)
	userID := uuid.New()
	usecase := &controllerUsecaseFake{}
	router := testRouter(usecase, authenticator)
	body := `{"full_name":"Updated Name","user_id":"` + uuid.NewString() + `"}`
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/profiles/me", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+signedToken(t, privateKey, userID, "profile:read"))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if usecase.updatedUserID != userID {
		t.Fatalf("updated user ID = %s, want verified token subject %s", usecase.updatedUserID, userID)
	}
}

func TestUploadAvatarAcceptsLocalStorageConfiguration(t *testing.T) {
	authenticator, privateKey := newTestAuthenticator(t)
	userID := uuid.New()
	router := gin.New()
	NewProfileController(
		&controllerUsecaseFake{},
		authenticator,
		config.AvatarConfig{LocalDir: t.TempDir(), MaxBytes: 1024},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	).Register(router)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("avatar", "avatar.png")
	if err != nil {
		t.Fatalf("create multipart avatar: %v", err)
	}
	if _, err := file.Write([]byte("avatar image")); err != nil {
		t.Fatalf("write multipart avatar: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/me/avatar", &body)
	request.Header.Set("Authorization", "Bearer "+signedToken(t, privateKey, userID, "profile:read"))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestAdminProfileReadRequiresRBACPermission(t *testing.T) {
	authenticator, privateKey := newTestAuthenticator(t)
	usecase := &controllerUsecaseFake{}
	router := testRouter(usecase, authenticator)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+uuid.NewString(), nil)
	request.Header.Set("Authorization", "Bearer "+signedToken(t, privateKey, uuid.New(), "profile.read"))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if usecase.publicReads != 0 {
		t.Fatal("admin read reached use case without RBAC permission")
	}
}

func TestAdminProfileReadRequiresGlobalAccessContext(t *testing.T) {
	authenticator, privateKey := newTestAuthenticator(t)
	usecase := &controllerUsecaseFake{}
	router := testRouter(usecase, authenticator)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+uuid.NewString(), nil)
	request.Header.Set("Authorization", "Bearer "+signedTokenWithScope(t, privateKey, uuid.New(), "profile.admin:read", "faculty"))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if usecase.publicReads != 0 {
		t.Fatal("admin read reached use case without a global access context")
	}
}

func TestAdminProfileReadDoesNotReturnPrivateData(t *testing.T) {
	authenticator, privateKey := newTestAuthenticator(t)
	usecase := &controllerUsecaseFake{}
	router := testRouter(usecase, authenticator)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+uuid.NewString(), nil)
	request.Header.Set("Authorization", "Bearer "+signedToken(t, privateKey, uuid.New(), "profile.admin:read"))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]any)
	if _, ok := data["phone_number"]; ok {
		t.Fatal("admin profile response exposed phone_number")
	}
	if _, ok := data["addresses"]; ok {
		t.Fatal("admin profile response exposed addresses")
	}
}

func testRouter(usecase *controllerUsecaseFake, authenticator *ummuauth.Auth) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewProfileController(usecase, authenticator, config.AvatarConfig{MaxBytes: 1024}, slog.New(slog.NewTextHandler(io.Discard, nil))).Register(router)
	return router
}

func newTestAuthenticator(t *testing.T) (*ummuauth.Auth, *rsa.PrivateKey) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	authenticator, err := ummuauth.New(ummuauth.Config{
		Issuer:      "https://auth.example.test",
		Audience:    "profile-service",
		KeyProvider: staticKeyProvider{key: &privateKey.PublicKey},
	})
	if err != nil {
		t.Fatalf("create authenticator: %v", err)
	}
	return authenticator, privateKey
}

func signedToken(t *testing.T, privateKey *rsa.PrivateKey, userID uuid.UUID, permission string) string {
	return signedTokenWithScope(t, privateKey, userID, permission, "global")
}

func signedTokenWithScope(t *testing.T, privateKey *rsa.PrivateKey, userID uuid.UUID, permission, scopeType string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":            "https://auth.example.test",
		"sub":            userID.String(),
		"aud":            "profile-service",
		"iat":            time.Now().Unix(),
		"exp":            time.Now().Add(time.Minute).Unix(),
		"jti":            uuid.NewString(),
		"principal_type": "user",
		"permissions":    strings.Fields(permission),
		"roles":          []string{"student"},
		"access_contexts": []map[string]any{
			{
				"role":        "super_admin",
				"scope":       map[string]any{"type": scopeType, "id": nil},
				"permissions": strings.Fields(permission),
			},
		},
	})
	token.Header["kid"] = "test-key"
	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return signed
}
