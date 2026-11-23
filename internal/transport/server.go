package transport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	ummuauth "github.com/AgungDev/ummu-go-auth"
	"github.com/AgungDev/ummu-profile-service/internal/config"
	"github.com/AgungDev/ummu-profile-service/internal/repositories"
	"github.com/AgungDev/ummu-profile-service/internal/services"
	profilehttp "github.com/AgungDev/ummu-profile-service/internal/transport/http"
	"github.com/AgungDev/ummu-profile-service/internal/usecases"
	"github.com/AgungDev/ummu-profile-service/migrations"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Server struct {
	httpServer *http.Server
	db         *gorm.DB
	logger     *slog.Logger
}

func New(ctx context.Context, cfg *config.Config) (*Server, error) {
	db, err := config.ConnectDB(cfg)
	if err != nil {
		return nil, err
	}
	if err := migrations.Apply(ctx, db); err != nil {
		return nil, fmt.Errorf("apply database migrations: %w", err)
	}

	authenticator, err := ummuauth.New(ummuauth.Config{
		Issuer:               cfg.AuthIssuer,
		JWKSURL:              cfg.JWKSURL,
		JWKSAllowedHTTPHosts: cfg.JWKSAllowedHTTPHosts,
		Audience:             cfg.AuthAudience,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize UMMU Go Auth: %w", err)
	}
	avatarStore, err := services.NewAvatarStore(cfg.Avatar)
	if err != nil {
		return nil, err
	}

	logger := slog.Default().With("service", cfg.AppName, "version", cfg.AppVersion)
	repository := repositories.NewProfileRepository(db)
	profileUsecase := usecases.NewProfileUsecase(repository, avatarStore, cfg.Avatar, logger)

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery())
	if cfg.Avatar.LocalDir != "" {
		if err := registerLocalAvatarFiles(router, cfg.Avatar.LocalDir); err != nil {
			return nil, err
		}
	}
	profilehttp.NewProfileController(profileUsecase, authenticator, cfg.Avatar, logger).Register(router)

	return &Server{
		httpServer: &http.Server{
			Addr:              ":" + cfg.AppPort,
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		db:     db,
		logger: logger,
	}, nil
}

func registerLocalAvatarFiles(router *gin.Engine, directory string) error {
	root, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("resolve local avatar directory: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("resolve local avatar directory: %w", err)
	}

	router.GET("/api/v1/profiles/avatars/*filepath", func(ctx *gin.Context) {
		key := strings.TrimPrefix(ctx.Param("filepath"), "/")
		cleanKey := filepath.Clean(filepath.FromSlash(key))
		if key == "" || filepath.IsAbs(cleanKey) || cleanKey == "." || cleanKey == ".." ||
			strings.HasPrefix(cleanKey, ".."+string(filepath.Separator)) {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}

		path := filepath.Join(root, cleanKey)
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}
		relative, err := filepath.Rel(root, resolved)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}
		info, err := os.Stat(resolved)
		if err != nil || info.IsDir() {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}
		extension := strings.ToLower(filepath.Ext(resolved))
		if extension != ".jpg" && extension != ".png" {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}

		ctx.Header("Cache-Control", "public, max-age=86400")
		ctx.Header("X-Content-Type-Options", "nosniff")
		ctx.File(resolved)
	})
	return nil
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("profile service listening", "address", s.httpServer.Addr)
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shut down HTTP server: %w", err)
		}
		sqlDB, err := s.db.DB()
		if err != nil {
			return fmt.Errorf("get PostgreSQL connection pool: %w", err)
		}
		if err := sqlDB.Close(); err != nil {
			return fmt.Errorf("close PostgreSQL connection: %w", err)
		}
		return nil
	}
}
