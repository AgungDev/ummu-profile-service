package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName              string
	AppVersion           string
	AppEnv               string
	AppPort              string
	DBHost               string
	DBPort               string
	DBUser               string
	DBPassword           string
	DBName               string
	DBSSLMode            string
	AuthIssuer           string
	AuthAudience         string
	JWKSURL              string
	JWKSAllowedHTTPHosts []string
	Avatar               AvatarConfig
}

type AvatarConfig struct {
	LocalDir   string
	Endpoint   string
	Bucket     string
	AccessKey  string
	SecretKey  string
	PublicBase string
	UseSSL     bool
	MaxBytes   int64
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	cfg := &Config{
		AppName:              env("APP_NAME", "ummu-profile-service"),
		AppVersion:           env("APP_VERSION", "1.0.0"),
		AppEnv:               env("APP_ENV", "development"),
		AppPort:              env("APP_PORT", "9002"),
		DBHost:               env("DB_HOST", "localhost"),
		DBPort:               env("DB_PORT", "54322"),
		DBUser:               env("DB_USER", "profile"),
		DBPassword:           os.Getenv("DB_PASSWORD"),
		DBName:               env("DB_NAME", "ummu_profile"),
		DBSSLMode:            env("DB_SSLMODE", "disable"),
		AuthIssuer:           os.Getenv("AUTH_ISSUER"),
		AuthAudience:         env("AUTH_AUDIENCE", "profile-service"),
		JWKSURL:              os.Getenv("JWKS_URL"),
		JWKSAllowedHTTPHosts: splitList(os.Getenv("JWKS_ALLOWED_HTTP_HOSTS")),
		Avatar: AvatarConfig{
			LocalDir:   os.Getenv("AVATAR_STORAGE_DIR"),
			Endpoint:   os.Getenv("AVATAR_ENDPOINT"),
			Bucket:     os.Getenv("AVATAR_BUCKET"),
			AccessKey:  os.Getenv("AVATAR_ACCESS_KEY"),
			SecretKey:  os.Getenv("AVATAR_SECRET_KEY"),
			PublicBase: strings.TrimRight(os.Getenv("AVATAR_PUBLIC_BASE_URL"), "/"),
			MaxBytes:   5 * 1024 * 1024,
		},
	}

	if value := os.Getenv("AVATAR_USE_SSL"); value != "" {
		useSSL, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("AVATAR_USE_SSL must be a boolean")
		}
		cfg.Avatar.UseSSL = useSSL
	}
	if value := os.Getenv("AVATAR_MAX_BYTES"); value != "" {
		maxBytes, err := strconv.ParseInt(value, 10, 64)
		if err != nil || maxBytes < 1 || maxBytes > 20*1024*1024 {
			return nil, fmt.Errorf("AVATAR_MAX_BYTES must be between 1 and 20971520")
		}
		cfg.Avatar.MaxBytes = maxBytes
	}

	for name, value := range map[string]string{
		"DB_PASSWORD":   cfg.DBPassword,
		"AUTH_ISSUER":   cfg.AuthIssuer,
		"JWKS_URL":      cfg.JWKSURL,
		"AUTH_AUDIENCE": cfg.AuthAudience,
	} {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("%s is required", name)
		}
	}
	issuer, err := url.Parse(cfg.AuthIssuer)
	if err != nil || !issuer.IsAbs() || issuer.Host == "" || (issuer.Scheme != "http" && issuer.Scheme != "https") {
		return nil, fmt.Errorf("AUTH_ISSUER must be an absolute HTTP(S) URL")
	}
	jwksURL, err := url.Parse(cfg.JWKSURL)
	if err != nil || !jwksURL.IsAbs() || jwksURL.Host == "" || (jwksURL.Scheme != "http" && jwksURL.Scheme != "https") {
		return nil, fmt.Errorf("JWKS_URL must be an absolute HTTP(S) URL")
	}

	if cfg.Avatar.LocalDir != "" {
		if cfg.Avatar.Endpoint != "" || cfg.Avatar.Bucket != "" ||
			cfg.Avatar.AccessKey != "" || cfg.Avatar.SecretKey != "" {
			return nil, fmt.Errorf("AVATAR_STORAGE_DIR cannot be combined with S3 avatar settings")
		}
		if cfg.Avatar.PublicBase == "" {
			return nil, fmt.Errorf("AVATAR_PUBLIC_BASE_URL is required with AVATAR_STORAGE_DIR")
		}
	} else {
		avatarFields := []string{cfg.Avatar.Endpoint, cfg.Avatar.Bucket, cfg.Avatar.AccessKey, cfg.Avatar.SecretKey, cfg.Avatar.PublicBase}
		configured := 0
		for _, field := range avatarFields {
			if field != "" {
				configured++
			}
		}
		if configured != 0 && configured != len(avatarFields) {
			return nil, fmt.Errorf("AVATAR_ENDPOINT, AVATAR_BUCKET, AVATAR_ACCESS_KEY, AVATAR_SECRET_KEY, and AVATAR_PUBLIC_BASE_URL must all be configured together")
		}
	}
	if cfg.Avatar.PublicBase != "" {
		publicURL, err := url.Parse(cfg.Avatar.PublicBase)
		if err != nil || !publicURL.IsAbs() || publicURL.Host == "" || (publicURL.Scheme != "http" && publicURL.Scheme != "https") {
			return nil, fmt.Errorf("AVATAR_PUBLIC_BASE_URL must be an absolute HTTP(S) URL")
		}
	}
	return cfg, nil
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if host := strings.TrimSpace(part); host != "" {
			values = append(values, host)
		}
	}
	return values
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
