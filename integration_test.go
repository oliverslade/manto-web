package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/manto/manto-web/internal/config"
	"github.com/manto/manto-web/internal/handlers"
	"github.com/manto/manto-web/internal/middleware/security"
	"github.com/manto/manto-web/internal/services"
)

func setupTestServer(_ *testing.T) *httptest.Server {
	cfg := &config.Config{}

	cfg.Server.Port = 8080
	cfg.Security.EnableHSTS = true
	cfg.Security.AllowedAPIEndpoints = []string{"https://api.anthropic.com"}
	cfg.Security.APIKeyMinLength = 10
	cfg.Logging.Level = "info"
	cfg.Anthropic.BaseURL = "https://api.anthropic.com"
	cfg.Anthropic.APIVersion = "2023-06-01"
	cfg.Anthropic.KeyPrefix = "sk-ant-"
	cfg.Anthropic.DefaultModel = "claude-3-5-haiku"
	cfg.Anthropic.MaxTokens = 1024
	cfg.Anthropic.Temperature = 0.7
	cfg.Anthropic.SystemMessage = "Be concise in your responses unless asked otherwise. Prefer tables and short paragraphs."
	cfg.Validation.MaxMessageLength = 4000

	anthropicService := services.NewAnthropicService(cfg)

	apiHandlers := handlers.NewAPIHandlers(cfg, anthropicService)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(security.SecurityHeaders(cfg))

	r.Get("/config.js", apiHandlers.ConfigHandler)
	r.Get("/api/models", apiHandlers.ModelsHandler)
	r.Post("/api/messages", apiHandlers.MessagesHandler)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	return httptest.NewServer(r)
}

func TestApplicationBehaviorIntegration(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	t.Run("health check works", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/healthz")
		if err != nil {
			t.Fatalf("health check failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", resp.StatusCode)
		}
	})

	t.Run("security headers are applied", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/config.js")
		if err != nil {
			t.Fatalf("config request failed: %v", err)
		}
		defer resp.Body.Close()

		expectedHeaders := map[string]string{
			"X-Content-Type-Options":       "nosniff",
			"X-Frame-Options":              "DENY",
			"Referrer-Policy":              "no-referrer",
			"Cross-Origin-Opener-Policy":   "same-origin",
			"Cross-Origin-Resource-Policy": "same-site",
			"Cross-Origin-Embedder-Policy": "require-corp",
			"Strict-Transport-Security":    "max-age=31536000; includeSubDomains; preload",
		}

		for header, expectedValue := range expectedHeaders {
			actualValue := resp.Header.Get(header)
			if actualValue != expectedValue {
				t.Errorf("expected header %s: %s, got: %s", header, expectedValue, actualValue)
			}
		}

		csp := resp.Header.Get("Content-Security-Policy")
		expectedCSPDirectives := []string{
			"default-src 'self'",
			"connect-src 'self' https://api.anthropic.com",
			"script-src 'self'",
		}

		for _, directive := range expectedCSPDirectives {
			if !strings.Contains(csp, directive) {
				t.Errorf("CSP should contain %s, got: %s", directive, csp)
			}
		}
	})

	t.Run("config endpoint provides consistent data", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/config.js")
		if err != nil {
			t.Fatalf("config request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}

		bodyStr := string(body)

		if !strings.HasPrefix(bodyStr, "window.MantoConfig = ") {
			t.Error("response should start with 'window.MantoConfig = '")
		}

		jsonStart := strings.Index(bodyStr, "{")
		jsonEnd := strings.LastIndex(bodyStr, "}")
		if jsonStart == -1 || jsonEnd == -1 {
			t.Fatal("could not find JSON in response")
		}

		jsonStr := bodyStr[jsonStart : jsonEnd+1]
		var configData map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &configData); err != nil {
			t.Fatalf("failed to parse config JSON: %v", err)
		}

		expectedSections := []string{"api", "providers", "validation", "models", "version"}
		for _, section := range expectedSections {
			if _, ok := configData[section]; !ok {
				t.Errorf("config should contain %s section", section)
			}
		}

		api, _ := configData["api"].(map[string]interface{})
		if api["anthropicKeyPrefix"] != "sk-ant-" {
			t.Errorf("expected anthropicKeyPrefix 'sk-ant-', got %v", api["anthropicKeyPrefix"])
		}

		validation, _ := configData["validation"].(map[string]interface{})
		if validation["minApiKeyLength"] != float64(10) {
			t.Errorf("expected minApiKeyLength 10, got %v", validation["minApiKeyLength"])
		}
	})

	t.Run("API endpoints validate requests properly", func(t *testing.T) {
		testCases := []struct {
			name           string
			method         string
			path           string
			headers        map[string]string
			body           string
			expectedStatus int
			expectedError  string
		}{
			{
				name:           "messages without API key",
				method:         "POST",
				path:           "/api/messages",
				body:           `{"model":"claude-3-haiku","messages":[{"role":"user","content":"hello"}],"max_tokens":100}`,
				expectedStatus: http.StatusBadRequest,
				expectedError:  "API key required",
			},
			{
				name:           "models without API key",
				method:         "GET",
				path:           "/api/models",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "API key required",
			},
			{
				name:   "messages with short API key",
				method: "POST",
				path:   "/api/messages",
				headers: map[string]string{
					"x-api-key": "short",
				},
				body:           `{"model":"claude-3-haiku","messages":[{"role":"user","content":"hello"}],"max_tokens":100}`,
				expectedStatus: http.StatusBadRequest,
				expectedError:  "Invalid API key format",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var bodyReader io.Reader
				if tc.body != "" {
					bodyReader = strings.NewReader(tc.body)
				}

				req, err := http.NewRequest(tc.method, server.URL+tc.path, bodyReader)
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}

				for key, value := range tc.headers {
					req.Header.Set(key, value)
				}

				resp, err := client.Do(req)
				if err != nil {
					t.Fatalf("request failed: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != tc.expectedStatus {
					t.Errorf("expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
				}

				if tc.expectedError != "" {
					body, _ := io.ReadAll(resp.Body)
					var errorResp map[string]string
					if err := json.Unmarshal(body, &errorResp); err != nil {
						t.Errorf("failed to parse error response: %v", err)
					} else if !strings.Contains(errorResp["error"], tc.expectedError) {
						t.Errorf("expected error containing '%s', got '%s'", tc.expectedError, errorResp["error"])
					}
				}
			})
		}
	})

	t.Run("middleware chain works correctly", func(t *testing.T) {
		start := time.Now()
		resp, err := client.Get(server.URL + "/config.js")
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		resp.Body.Close()

		if duration > 30*time.Second {
			t.Error("request took too long, middleware timeout may not be working")
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected successful response, got %d", resp.StatusCode)
		}
	})
}

func TestConfigurationConsistency(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	t.Run("config endpoint and validation are consistent", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/config.js")
		if err != nil {
			t.Fatalf("config request failed: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		jsonStart := strings.Index(bodyStr, "{")
		jsonEnd := strings.LastIndex(bodyStr, "}")
		jsonStr := bodyStr[jsonStart : jsonEnd+1]

		var configData map[string]interface{}
		json.Unmarshal([]byte(jsonStr), &configData)

		validation := configData["validation"].(map[string]interface{})
		minApiKeyLength := int(validation["minApiKeyLength"].(float64))

		shortKey := strings.Repeat("x", minApiKeyLength-1)
		validLengthKey := "sk-ant-" + strings.Repeat("x", minApiKeyLength)

		req, _ := http.NewRequest("POST", server.URL+"/api/messages",
			bytes.NewBufferString(`{"model":"test","messages":[{"role":"user","content":"hi"}],"max_tokens":100}`))
		req.Header.Set("x-api-key", shortKey)

		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Error("short API key should be rejected")
		}

		req, _ = http.NewRequest("POST", server.URL+"/api/messages",
			bytes.NewBufferString(`{"model":"test","messages":[{"role":"user","content":"hi"}],"max_tokens":100}`))
		req.Header.Set("x-api-key", validLengthKey)

		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusBadRequest {
			body, _ := io.ReadAll(resp.Body)
			var errorResp map[string]string
			json.Unmarshal(body, &errorResp)
			if strings.Contains(errorResp["error"], "Invalid API key format") {
				t.Error("valid length key should not fail format validation")
			}
		}
	})
}
