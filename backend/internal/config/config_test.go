package config

import (
	"testing"
)

func TestLoad_CookieSecret_Validation(t *testing.T) {
	t.Run("development allows missing secret", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		t.Setenv("COOKIE_SECRET", "")
		cfg := Load()
		if cfg.CookieSecret == "" {
			t.Fatalf("expected dev default secret to be applied")
		}
		if cfg.CookieSecret != "dev-insecure-change-in-production" {
			t.Fatalf("unexpected dev secret: %s", cfg.CookieSecret)
		}
	})

	t.Run("production requires secret", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("COOKIE_SECRET", "")
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic for missing COOKIE_SECRET")
			}
		}()
		Load()
	})

	t.Run("production rejects dev default", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("COOKIE_SECRET", "dev-insecure-change-in-production")
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic for dev default in production")
			}
		}()
		Load()
	})

	t.Run("production accepts real secret", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("COOKIE_SECRET", "real-secret-123")
		t.Setenv("ENFORCE_DEVICE_FINGERPRINT", "true")
		cfg := Load()
		if cfg.CookieSecret != "real-secret-123" {
			t.Fatalf("secret not loaded")
		}
	})
}

func TestLoad_DeviceFingerprint_Enforcement(t *testing.T) {
	t.Run("development allows enforcement disabled", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		t.Setenv("ENFORCE_DEVICE_FINGERPRINT", "")
		t.Setenv("COOKIE_SECRET", "")
		cfg := Load()
		if cfg.EnforceDeviceFingerprint {
			t.Fatalf("expected enforcement to be false in development by default")
		}
	})

	t.Run("production requires enforcement enabled", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("COOKIE_SECRET", "real-secret")
		t.Setenv("ENFORCE_DEVICE_FINGERPRINT", "")
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic when ENFORCE_DEVICE_FINGERPRINT is false in production")
			}
		}()
		Load()
	})

	t.Run("production accepts enforcement enabled", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("COOKIE_SECRET", "real-secret")
		t.Setenv("ENFORCE_DEVICE_FINGERPRINT", "true")
		cfg := Load()
		if !cfg.EnforceDeviceFingerprint {
			t.Fatalf("expected enforcement to be true")
		}
	})
}
