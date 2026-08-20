package config

import (
	"os"
)

type Config struct {
	Port               string
	DatabaseURL        string
	LogLevel           string
	AppEnv             string
	CORSAllowedOrigins string

	JWTSecret          string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	FrontendURL        string

	// Avatar storage (S3-compatible, e.g. Contabo Object Storage).
	// If S3AvatarBucket is empty the API falls back to local /var/www/files/avatars.
	S3AvatarBucket        string
	S3AvatarRegion        string
	S3AvatarEndpoint      string
	S3AvatarPublicBaseURL string
	S3AvatarAccessKeyID   string
	S3AvatarSecretKey     string
}

func Load() Config {
	return Config{
		Port:               envOrDefault("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		LogLevel:           envOrDefault("LOG_LEVEL", "info"),
		AppEnv:             envOrDefault("APP_ENV", "development"),
		CORSAllowedOrigins: envOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000"),

		JWTSecret:          os.Getenv("JWT_SECRET"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  envOrDefault("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback"),
		FrontendURL:        envOrDefault("FRONTEND_URL", "http://localhost:5173"),

		S3AvatarBucket:        os.Getenv("AVATAR_S3_BUCKET"),
		S3AvatarRegion:        os.Getenv("AVATAR_S3_REGION"),
		S3AvatarEndpoint:      os.Getenv("AVATAR_S3_ENDPOINT"),
		S3AvatarPublicBaseURL: os.Getenv("AVATAR_S3_PUBLIC_BASE_URL"),
		S3AvatarAccessKeyID:   os.Getenv("AVATAR_S3_ACCESS_KEY_ID"),
		S3AvatarSecretKey:     os.Getenv("AVATAR_S3_SECRET_ACCESS_KEY"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
