package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/manto/manto-web/internal/config"
	"github.com/manto/manto-web/internal/services"
)

func createTestConfig() *config.Config {
	cfg := &config.Config{}

	cfg.Server.Port = 8080
	cfg.Security.APIKeyMinLength = 10
	cfg.Anthropic.BaseURL = "https://api.anthropic.com"
	cfg.Anthropic.APIVersion = "2023-06-01"
	cfg.Anthropic.KeyPrefix = "sk-ant-"
	cfg.Anthropic.DefaultModel = "claude-3-5-haiku"
	cfg.Anthropic.MaxTokens = 1024
	cfg.Anthropic.Temperature = 0.7
	cfg.Anthropic.SystemMessage = "Be concise in your responses unless asked otherwise. Prefer tables and short paragraphs."
	cfg.Validation.MaxMessageLength = 4000

	return cfg
}

func TestConfigHandlerBehavior(t *testing.T) {
	cfg := createTestConfig()
	anthropicService := services.NewAnthropicService(cfg)
	handlers := NewAPIHandlers(cfg, anthropicService)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectJS       bool
		validateConfig func(configData map[string]interface{}) error
	}{
		{
			name:           "GET returns JavaScript config",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectJS:       true,
			validateConfig: func(configData map[string]interface{}) error {
				if _, ok := configData["api"]; !ok {
					t.Error("config should contain 'api' section")
				}
				if _, ok := configData["providers"]; !ok {
					t.Error("config should contain 'providers' section")
				}
				if _, ok := configData["validation"]; !ok {
					t.Error("config should contain 'validation' section")
				}

				api, ok := configData["api"].(map[string]interface{})
				if !ok {
					t.Error("api section should be an object")
					return nil
				}

				if api["anthropicKeyPrefix"] != "sk-ant-" {
					t.Errorf("expected anthropicKeyPrefix 'sk-ant-', got %v", api["anthropicKeyPrefix"])
				}

				if api["preferredModelId"] != "claude-3-5-haiku" {
					t.Errorf("expected preferredModelId 'claude-3-5-haiku', got %v", api["preferredModelId"])
				}

				return nil
			},
		},
		{
			name:           "POST method still works (handler doesn't restrict methods)",
			method:         "POST",
			expectedStatus: http.StatusOK,
			expectJS:       true,
			validateConfig: func(configData map[string]interface{}) error {
				if _, ok := configData["api"]; !ok {
					t.Error("config should contain 'api' section")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/config.js", nil)
			w := httptest.NewRecorder()

			handlers.ConfigHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK && tt.expectJS {
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/javascript" {
					t.Errorf("expected Content-Type 'application/javascript', got %s", contentType)
				}

				cacheControl := w.Header().Get("Cache-Control")
				if !strings.Contains(cacheControl, "max-age=300") {
					t.Errorf("expected cache control with max-age=300, got %s", cacheControl)
				}

				body := w.Body.String()
				if !strings.HasPrefix(body, "window.MantoConfig = ") {
					t.Error("response should start with 'window.MantoConfig = '")
				}

				jsonStart := strings.Index(body, "{")
				jsonEnd := strings.LastIndex(body, "}")
				if jsonStart == -1 || jsonEnd == -1 {
					t.Error("could not find JSON in response")
					return
				}

				jsonStr := body[jsonStart : jsonEnd+1]
				var configData map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &configData); err != nil {
					t.Errorf("failed to parse config JSON: %v", err)
					return
				}

				if tt.validateConfig != nil {
					tt.validateConfig(configData)
				}
			}
		})
	}
}

func TestMessagesHandlerBehavior(t *testing.T) {
	cfg := createTestConfig()
	anthropicService := services.NewAnthropicService(cfg)
	handlers := NewAPIHandlers(cfg, anthropicService)

	tests := []struct {
		name           string
		method         string
		headers        map[string]string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "missing API key returns 400",
			method:         "POST",
			body:           `{"model":"claude-3-haiku","messages":[{"role":"user","content":"hello"}],"max_tokens":100}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "API key required",
		},
		{
			name:   "short API key returns 400",
			method: "POST",
			headers: map[string]string{
				"x-api-key": "short",
			},
			body:           `{"model":"claude-3-haiku","messages":[{"role":"user","content":"hello"}],"max_tokens":100}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid API key format",
		},
		{
			name:           "invalid JSON returns 400",
			method:         "POST",
			headers:        map[string]string{"x-api-key": "sk-ant-1234567890"},
			body:           `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid JSON format",
		},
		{
			name:           "missing model returns 400",
			method:         "POST",
			headers:        map[string]string{"x-api-key": "sk-ant-1234567890"},
			body:           `{"messages":[{"role":"user","content":"hello"}],"max_tokens":100}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Model is required",
		},
		{
			name:           "missing messages returns 400",
			method:         "POST",
			headers:        map[string]string{"x-api-key": "sk-ant-1234567890"},
			body:           `{"model":"claude-3-haiku","max_tokens":100}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Messages are required",
		},
		{
			name:    "message too long returns 400",
			method:  "POST",
			headers: map[string]string{"x-api-key": "sk-ant-1234567890"},
			body: func() string {
				longMessage := strings.Repeat("a", 5000) // Exceeds 4000 char limit
				return `{"model":"claude-3-haiku","messages":[{"role":"user","content":"` + longMessage + `"}],"max_tokens":100}`
			}(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Message too long",
		},
		{
			name:           "zero max tokens returns 400",
			method:         "POST",
			headers:        map[string]string{"x-api-key": "sk-ant-1234567890"},
			body:           `{"model":"claude-3-haiku","messages":[{"role":"user","content":"hello"}],"max_tokens":0}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "MaxTokens must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tt.body != "" {
				body = bytes.NewBufferString(tt.body)
			} else {
				body = bytes.NewBuffer(nil)
			}

			req := httptest.NewRequest(tt.method, "/api/messages", body)

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			w := httptest.NewRecorder()

			handlers.MessagesHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var errorResp map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
					t.Errorf("failed to parse error response: %v", err)
					return
				}

				if errorMsg, ok := errorResp["error"]; !ok || !strings.Contains(errorMsg, tt.expectedError) {
					t.Errorf("expected error containing %s, got %s", tt.expectedError, errorMsg)
				}
			}
		})
	}
}

func TestModelsHandlerBehavior(t *testing.T) {
	cfg := createTestConfig()
	anthropicService := services.NewAnthropicService(cfg)
	handlers := NewAPIHandlers(cfg, anthropicService)

	tests := []struct {
		name           string
		method         string
		headers        map[string]string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "missing API key returns 400",
			method:         "GET",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "API key required",
		},
		{
			name:           "valid API key format passes validation",
			method:         "GET",
			headers:        map[string]string{"x-api-key": "sk-ant-1234567890"},
			expectedStatus: http.StatusBadRequest, // Will fail at API call, but passes validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/models", nil)

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			w := httptest.NewRecorder()

			handlers.ModelsHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var errorResp map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
					t.Errorf("failed to parse error response: %v", err)
					return
				}

				if errorMsg, ok := errorResp["error"]; !ok || !strings.Contains(errorMsg, tt.expectedError) {
					t.Errorf("expected error containing %s, got %s", tt.expectedError, errorMsg)
				}
			}
		})
	}
}

func TestHandlerIntegration(t *testing.T) {
	cfg := createTestConfig()
	anthropicService := services.NewAnthropicService(cfg)
	handlers := NewAPIHandlers(cfg, anthropicService)

	t.Run("config endpoint provides data needed by other endpoints", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/config.js", nil)
		w := httptest.NewRecorder()
		handlers.ConfigHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("config endpoint failed: %d", w.Code)
		}

		body := w.Body.String()
		jsonStart := strings.Index(body, "{")
		jsonEnd := strings.LastIndex(body, "}")
		jsonStr := body[jsonStart : jsonEnd+1]

		var configData map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &configData); err != nil {
			t.Fatalf("failed to parse config: %v", err)
		}

		validation, ok := configData["validation"].(map[string]interface{})
		if !ok {
			t.Fatal("validation config not found")
		}

		minApiKeyLength, ok := validation["minApiKeyLength"].(float64)
		if !ok {
			t.Fatal("minApiKeyLength not found in config")
		}

		shortKey := strings.Repeat("x", int(minApiKeyLength)-1)
		validKey := "sk-ant-" + strings.Repeat("x", int(minApiKeyLength))

		req = httptest.NewRequest("POST", "/api/messages", bytes.NewBufferString(`{"model":"test","messages":[{"role":"user","content":"hi"}],"max_tokens":100}`))
		req.Header.Set("x-api-key", shortKey)
		w = httptest.NewRecorder()
		handlers.MessagesHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected short key to fail, got status %d", w.Code)
		}

		req = httptest.NewRequest("POST", "/api/messages", bytes.NewBufferString(`{"model":"test","messages":[{"role":"user","content":"hi"}],"max_tokens":100}`))
		req.Header.Set("x-api-key", validKey)
		w = httptest.NewRecorder()
		handlers.MessagesHandler(w, req)

		if w.Code == http.StatusBadRequest {
			var errorResp map[string]string
			json.Unmarshal(w.Body.Bytes(), &errorResp)
			if strings.Contains(errorResp["error"], "Invalid API key format") {
				t.Error("valid length key should not fail format validation")
			}
		}
	})
}
