package services

import (
	"testing"

	"github.com/manto/manto-web/internal/config"
)

func createTestConfig() *config.Config {
	cfg := &config.Config{}

	cfg.Anthropic.BaseURL = "https://api.anthropic.com"
	cfg.Anthropic.APIVersion = "2023-06-01"
	cfg.Anthropic.KeyPrefix = "sk-ant-"
	cfg.Anthropic.DefaultModel = "claude-3-5-haiku"
	cfg.Anthropic.MaxTokens = 1024
	cfg.Anthropic.Temperature = 0.7
	cfg.Security.APIKeyMinLength = 10

	return cfg
}

func TestAPIKeyValidationBehavior(t *testing.T) {
	cfg := createTestConfig()
	service := NewAnthropicService(cfg)

	tests := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{
			name:     "valid API key with correct prefix and length",
			apiKey:   "sk-ant-1234567890abcdef",
			expected: true,
		},
		{
			name:     "too short API key",
			apiKey:   "sk-ant-12",
			expected: false,
		},
		{
			name:     "wrong prefix",
			apiKey:   "sk-openai-1234567890abcdef",
			expected: false,
		},
		{
			name:     "no prefix",
			apiKey:   "1234567890abcdef",
			expected: false,
		},
		{
			name:     "empty key",
			apiKey:   "",
			expected: false,
		},
		{
			name:     "exactly minimum length with correct prefix",
			apiKey:   "sk-ant-123", // prefix + 3 chars = 10 total
			expected: true,
		},
		{
			name:     "one character too short",
			apiKey:   "sk-ant-12", // prefix + 2 chars = 9 total
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ValidateAPIKey(tt.apiKey)
			if result != tt.expected {
				t.Errorf("expected %v for key %s, got %v", tt.expected, tt.apiKey, result)
			}
		})
	}
}

func TestServiceConfigurationBehavior(t *testing.T) {
	tests := []struct {
		name         string
		modifyConfig func(*config.Config)
		testBehavior func(*testing.T, *AnthropicService)
	}{
		{
			name: "service respects custom API key prefix",
			modifyConfig: func(cfg *config.Config) {
				cfg.Anthropic.KeyPrefix = "custom-prefix-"
				cfg.Security.APIKeyMinLength = 15
			},
			testBehavior: func(t *testing.T, service *AnthropicService) {
				validKey := "custom-prefix-12345"
				if !service.ValidateAPIKey(validKey) {
					t.Error("should accept key with custom prefix")
				}

				oldPrefixKey := "sk-ant-12345678901234"
				if service.ValidateAPIKey(oldPrefixKey) {
					t.Error("should reject key with old prefix")
				}
			},
		},
		{
			name: "service respects custom minimum key length",
			modifyConfig: func(cfg *config.Config) {
				cfg.Security.APIKeyMinLength = 20
			},
			testBehavior: func(t *testing.T, service *AnthropicService) {
				shortKey := "sk-ant-1234567890" // 17 chars, less than 20
				if service.ValidateAPIKey(shortKey) {
					t.Error("should reject key shorter than custom minimum")
				}

				longKey := "sk-ant-1234567890abcdef" // 23 chars, more than 20
				if !service.ValidateAPIKey(longKey) {
					t.Error("should accept key meeting custom minimum")
				}
			},
		},
		{
			name: "service uses configured base URL",
			modifyConfig: func(cfg *config.Config) {
				cfg.Anthropic.BaseURL = "https://custom-api.example.com"
			},
			testBehavior: func(t *testing.T, service *AnthropicService) {
				if service.config.Anthropic.BaseURL != "https://custom-api.example.com" {
					t.Error("service should store custom base URL")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig()
			tt.modifyConfig(cfg)
			service := NewAnthropicService(cfg)
			tt.testBehavior(t, service)
		})
	}
}

func TestServiceErrorHandlingBehavior(t *testing.T) {
	cfg := createTestConfig()
	service := NewAnthropicService(cfg)

	t.Run("GetModels with invalid URL returns error", func(t *testing.T) {
		originalURL := service.config.Anthropic.BaseURL
		service.config.Anthropic.BaseURL = "://invalid-url"
		defer func() { service.config.Anthropic.BaseURL = originalURL }()

		_, err := service.GetModels("sk-ant-validkey123")
		if err == nil {
			t.Error("expected error for invalid URL")
		}

		if err != nil && !containsAnyString(err.Error(), []string{"failed to create request", "invalid URL"}) {
			t.Errorf("expected URL-related error, got: %v", err)
		}
	})

	t.Run("SendMessage with invalid request returns error", func(t *testing.T) {
		_, err := service.SendMessage("sk-ant-validkey123", nil)
		if err == nil {
			t.Error("expected error for nil request")
		}
	})
}

func containsAnyString(text string, substrings []string) bool {
	for _, substring := range substrings {
		if len(text) >= len(substring) {
			for i := 0; i <= len(text)-len(substring); i++ {
				if text[i:i+len(substring)] == substring {
					return true
				}
			}
		}
	}
	return false
}
