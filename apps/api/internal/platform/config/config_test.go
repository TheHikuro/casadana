package config

import (
	"testing"
)

func TestLoad_ReadsRequiredEnv(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "u")
	t.Setenv("POSTGRES_PASSWORD", "p")
	t.Setenv("POSTGRES_DB", "casadana")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("RESEND_API_KEY", "re_test")
	t.Setenv("MAIL_FROM", "no-reply@casa-dana.com")
	t.Setenv("WEB_ORIGIN", "http://localhost:5173")
	t.Setenv("ADMIN_NOTIFY_EMAIL", "owner@casa-dana.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("default Port = %d, want 8080", cfg.Port)
	}
	if cfg.DB.Host != "localhost" || cfg.DB.Name != "casadana" {
		t.Errorf("DB config not parsed: %+v", cfg.DB)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("default LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
}

func TestLoad_FailsOnMissingRequired(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error from missing required env, got nil")
	}
}

func TestLoad_CookieSecureDefaultsTrue(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "u")
	t.Setenv("POSTGRES_PASSWORD", "p")
	t.Setenv("POSTGRES_DB", "casadana")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("RESEND_API_KEY", "re_test")
	t.Setenv("MAIL_FROM", "no-reply@casadana.com")
	t.Setenv("WEB_ORIGIN", "http://localhost:5173")
	t.Setenv("ADMIN_NOTIFY_EMAIL", "owner@casadana.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CookieSecure {
		t.Error("CookieSecure default = false, want true")
	}
}

func TestLoad_CookieSecureFalseOverride(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "u")
	t.Setenv("POSTGRES_PASSWORD", "p")
	t.Setenv("POSTGRES_DB", "casadana")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("RESEND_API_KEY", "re_test")
	t.Setenv("MAIL_FROM", "no-reply@casadana.com")
	t.Setenv("WEB_ORIGIN", "http://localhost:5173")
	t.Setenv("ADMIN_NOTIFY_EMAIL", "owner@casadana.com")
	t.Setenv("COOKIE_SECURE", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CookieSecure {
		t.Error("CookieSecure = true, want false after override")
	}
}
