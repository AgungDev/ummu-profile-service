package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/AgungDev/ummu-profile-service/internal/config"
	"github.com/AgungDev/ummu-profile-service/migrations"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type legacyUser struct {
	ID       uuid.UUID
	FullName string
}

func (legacyUser) TableName() string { return "users" }

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("load .env: %v", err)
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}
	profileDB, err := config.ConnectDB(cfg)
	if err != nil {
		log.Fatalf("connect to Profile Service database: %v", err)
	}
	if err := migrations.Apply(context.Background(), profileDB); err != nil {
		log.Fatalf("apply Profile Service migrations: %v", err)
	}

	authDSN, err := legacyAuthDSN()
	if err != nil {
		log.Fatal(err)
	}
	authDB, err := gorm.Open(postgres.Open(authDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect to legacy Auth Service database: %v", err)
	}

	var users []legacyUser
	if err := authDB.Where("full_name IS NOT NULL AND BTRIM(full_name) <> ''").Find(&users).Error; err != nil {
		log.Fatalf("read legacy Auth Service names: %v", err)
	}
	imported := 0
	for _, user := range users {
		result := profileDB.Exec(`
			INSERT INTO user_profiles (user_id, full_name)
			VALUES (?, ?)
			ON CONFLICT (user_id) DO UPDATE
			SET full_name = EXCLUDED.full_name, updated_at = NOW()
			WHERE user_profiles.full_name = ''
		`, user.ID, strings.TrimSpace(user.FullName))
		if result.Error != nil {
			log.Fatalf("import user profile %s: %v", user.ID, result.Error)
		}
		if result.RowsAffected > 0 {
			imported++
		}
	}
	fmt.Printf("Imported or populated %d profiles (source records: %d).\n", imported, len(users))
}

func legacyAuthDSN() (string, error) {
	host := strings.TrimSpace(os.Getenv("LEGACY_AUTH_DB_HOST"))
	port := strings.TrimSpace(os.Getenv("LEGACY_AUTH_DB_PORT"))
	user := os.Getenv("LEGACY_AUTH_DB_USER")
	password := os.Getenv("LEGACY_AUTH_DB_PASSWORD")
	name := strings.TrimSpace(os.Getenv("LEGACY_AUTH_DB_NAME"))
	sslmode := strings.TrimSpace(os.Getenv("LEGACY_AUTH_DB_SSLMODE"))
	if host == "" || port == "" || user == "" || password == "" || name == "" {
		return "", fmt.Errorf("LEGACY_AUTH_DB_HOST, LEGACY_AUTH_DB_PORT, LEGACY_AUTH_DB_USER, LEGACY_AUTH_DB_PASSWORD, and LEGACY_AUTH_DB_NAME are required")
	}
	if sslmode == "" {
		sslmode = "disable"
	}
	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   name,
	}
	query := dsn.Query()
	query.Set("sslmode", sslmode)
	dsn.RawQuery = query.Encode()
	return dsn.String(), nil
}
