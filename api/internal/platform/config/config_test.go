package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("expected Port=8080, got %s", cfg.Port)
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("expected DatabaseURL empty, got %s", cfg.DatabaseURL)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel=info, got %s", cfg.LogLevel)
	}
	if cfg.AppEnv != "development" {
		t.Errorf("expected AppEnv=development, got %s", cfg.AppEnv)
	}
	if cfg.CORSAllowedOrigins != "http://localhost:5173,http://localhost:3000" {
		t.Errorf("expected CORSAllowedOrigins default, got %s", cfg.CORSAllowedOrigins)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("PORT", "3000")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("APP_ENV", "production")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.task-tracker.dev")

	cfg := Load()

	if cfg.Port != "3000" {
		t.Errorf("expected Port=3000, got %s", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://localhost/test" {
		t.Errorf("expected DatabaseURL=postgres://localhost/test, got %s", cfg.DatabaseURL)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel=debug, got %s", cfg.LogLevel)
	}
	if cfg.AppEnv != "production" {
		t.Errorf("expected AppEnv=production, got %s", cfg.AppEnv)
	}
	if cfg.CORSAllowedOrigins != "https://app.task-tracker.dev" {
		t.Errorf("expected CORSAllowedOrigins=https://app.task-tracker.dev, got %s", cfg.CORSAllowedOrigins)
	}
}
