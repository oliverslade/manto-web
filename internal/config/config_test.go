package config

import (
	"os"
	"testing"
	"time"
)

func TestConfigLoadBehavior(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		setupFiles  func(tempDir string)
		cleanupEnv  func()
		expectError bool
		validate    func(*Config) error
	}{
		{
			name: "loads defaults when no env vars or files",
			setupEnv: func() {
				// Clear any existing env vars
				os.Unsetenv("GO_ENV")
				os.Unsetenv("PORT")
				os.Unsetenv("LOG_LEVEL")
			},
			setupFiles: func(tempDir string) {
				// No config files
			},
			cleanupEnv: func() {},
			validate: func(cfg *Config) error {
				if cfg.Server.Port != 8080 {
					t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
				}
				if cfg.Logging.Level != "info" {
					t.Errorf("expected default log level 'info', got %s", cfg.Logging.Level)
				}
				if cfg.Anthropic.DefaultModel != "claude-3-5-haiku" {
					t.Errorf("expected default model 'claude-3-5-haiku', got %s", cfg.Anthropic.DefaultModel)
				}
				return nil
			},
		},
		{
			name: "environment variables override defaults",
			setupEnv: func() {
				os.Setenv("PORT", "9999")
				os.Setenv("LOG_LEVEL", "debug")
				os.Setenv("ANTHROPIC_MAX_TOKENS", "2048")
			},
			setupFiles: func(tempDir string) {},
			cleanupEnv: func() {
				os.Unsetenv("PORT")
				os.Unsetenv("LOG_LEVEL")
				os.Unsetenv("ANTHROPIC_MAX_TOKENS")
			},
			validate: func(cfg *Config) error {
				if cfg.Server.Port != 9999 {
					t.Errorf("expected port 9999 from env, got %d", cfg.Server.Port)
				}
				if cfg.Logging.Level != "debug" {
					t.Errorf("expected log level 'debug' from env, got %s", cfg.Logging.Level)
				}
				if cfg.Anthropic.MaxTokens != 2048 {
					t.Errorf("expected max tokens 2048 from env, got %d", cfg.Anthropic.MaxTokens)
				}
				return nil
			},
		},
		{
			name: "validates configuration values",
			setupEnv: func() {
				os.Setenv("PORT", "99999") // Invalid port
			},
			setupFiles: func(tempDir string) {},
			cleanupEnv: func() {
				os.Unsetenv("PORT")
			},
			expectError: true,
		},
		{
			name: "handles duration parsing",
			setupEnv: func() {
				os.Setenv("READ_TIMEOUT", "45s")
				os.Setenv("ANTHROPIC_TIMEOUT", "2m")
			},
			setupFiles: func(tempDir string) {},
			cleanupEnv: func() {
				os.Unsetenv("READ_TIMEOUT")
				os.Unsetenv("ANTHROPIC_TIMEOUT")
			},
			validate: func(cfg *Config) error {
				if cfg.Server.ReadTimeout.Duration != 45*time.Second {
					t.Errorf("expected read timeout 45s, got %v", cfg.Server.ReadTimeout.Duration)
				}
				if cfg.Anthropic.Timeout.Duration != 2*time.Minute {
					t.Errorf("expected anthropic timeout 2m, got %v", cfg.Anthropic.Timeout.Duration)
				}
				return nil
			},
		},
		{
			name: "handles comma-separated lists",
			setupEnv: func() {
				os.Setenv("ALLOWED_API_ENDPOINTS", "https://api.anthropic.com,https://api.example.com")
				os.Setenv("ALLOWED_HOSTS", "localhost,example.com")
			},
			setupFiles: func(tempDir string) {},
			cleanupEnv: func() {
				os.Unsetenv("ALLOWED_API_ENDPOINTS")
				os.Unsetenv("ALLOWED_HOSTS")
			},
			validate: func(cfg *Config) error {
				expectedEndpoints := []string{"https://api.anthropic.com", "https://api.example.com"}
				if len(cfg.Security.AllowedAPIEndpoints) != 2 {
					t.Errorf("expected 2 endpoints, got %d", len(cfg.Security.AllowedAPIEndpoints))
				}
				for i, expected := range expectedEndpoints {
					if cfg.Security.AllowedAPIEndpoints[i] != expected {
						t.Errorf("expected endpoint %s, got %s", expected, cfg.Security.AllowedAPIEndpoints[i])
					}
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			cfg, err := Load()

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !tt.expectError && tt.validate != nil {
				if err := tt.validate(cfg); err != nil {
					t.Errorf("validation failed: %v", err)
				}
			}
		})
	}
}

func TestEnvironmentDetectionBehavior(t *testing.T) {
	tests := []struct {
		name        string
		goEnv       string
		environment string
		expectedEnv string
	}{
		{
			name:        "defaults to production when no env vars",
			expectedEnv: "production",
		},
		{
			name:        "uses GO_ENV when set",
			goEnv:       "development",
			expectedEnv: "development",
		},
		{
			name:        "falls back to ENVIRONMENT when GO_ENV not set",
			environment: "staging",
			expectedEnv: "staging",
		},
		{
			name:        "GO_ENV takes precedence over ENVIRONMENT",
			goEnv:       "development",
			environment: "staging",
			expectedEnv: "development",
		},
		{
			name:        "normalizes environment to lowercase",
			goEnv:       "PRODUCTION",
			expectedEnv: "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("GO_ENV")
			os.Unsetenv("ENVIRONMENT")

			if tt.goEnv != "" {
				os.Setenv("GO_ENV", tt.goEnv)
			}
			if tt.environment != "" {
				os.Setenv("ENVIRONMENT", tt.environment)
			}

			env := GetEnvironment()

			if env != tt.expectedEnv {
				t.Errorf("expected environment %s, got %s", tt.expectedEnv, env)
			}

			os.Unsetenv("GO_ENV")
			os.Unsetenv("ENVIRONMENT")
		})
	}
}
