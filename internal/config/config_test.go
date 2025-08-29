package config

import (
	"strings"
	"testing"
	"time"
)

func TestConfigLoadBehavior(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  func(tempDir string)
		expectError bool
		validate    func(*Config) error
	}{
		{
			name: "loads defaults when no env vars or files",
			setupFiles: func(tempDir string) {
				// No config files
			},
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
			name:       "environment variables override defaults",
			setupFiles: func(tempDir string) {},
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
			name:        "validates configuration values",
			setupFiles:  func(tempDir string) {},
			expectError: true,
		},
		{
			name:       "handles duration parsing",
			setupFiles: func(tempDir string) {},
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
			name:       "handles comma-separated lists",
			setupFiles: func(tempDir string) {},
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
			if strings.Contains(tt.name, "override") {
				t.Setenv("PORT", "9999")
				t.Setenv("LOG_LEVEL", "debug")
				t.Setenv("ANTHROPIC_MAX_TOKENS", "2048")
			}
			if strings.Contains(tt.name, "validates") {
				t.Setenv("PORT", "99999")
			}
			if strings.Contains(tt.name, "duration") {
				t.Setenv("READ_TIMEOUT", "45s")
				t.Setenv("ANTHROPIC_TIMEOUT", "2m")
			}
			if strings.Contains(tt.name, "comma-separated") {
				t.Setenv("ALLOWED_API_ENDPOINTS", "https://api.anthropic.com,https://api.example.com")
				t.Setenv("ALLOWED_HOSTS", "localhost,example.com")
			}

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
			if tt.goEnv != "" {
				t.Setenv("GO_ENV", tt.goEnv)
			}
			if tt.environment != "" {
				t.Setenv("ENVIRONMENT", tt.environment)
			}

			env := GetEnvironment()

			if env != tt.expectedEnv {
				t.Errorf("expected environment %s, got %s", tt.expectedEnv, env)
			}
		})
	}
}
