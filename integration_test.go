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

		expectedSections := []string{"api", "providers", "validation", "version"}
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

func TestAnthropicAPIIntegration(t *testing.T) {
	fakeAnthropic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("x-api-key")
		if apiKey == "sk-ant-invalid" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"type":    "authentication_error",
					"message": "Invalid API key",
				},
			})
			return
		}

		if r.URL.Path == "/v1/messages" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"id":      "msg_1",
				"type":    "message",
				"role":    "assistant",
				"content": []map[string]string{{"type": "text", "text": "Hello! How can I help you?"}},
				"model":   "claude-3-5-haiku",
				"stop_reason": "end_turn",
				"usage": map[string]int{
					"input_tokens":  5,
					"output_tokens": 3,
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.URL.Path == "/v1/models" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			models := map[string]interface{}{
				"data": []map[string]string{
					{"id": "claude-3-5-haiku", "name": "Claude 3.5 Haiku"},
					{"id": "claude-3-5-sonnet", "name": "Claude 3.5 Sonnet"},
				},
			}
			json.NewEncoder(w).Encode(models)
			return
		}

		http.NotFound(w, r)
	}))
	defer fakeAnthropic.Close()

	// Create config with fake Anthropic URL
	cfg := &config.Config{}
	cfg.Server.Port = 8080
	cfg.Server.ReadTimeout = config.Duration{Duration: 30 * time.Second}
	cfg.Server.WriteTimeout = config.Duration{Duration: 30 * time.Second}
	cfg.Security.EnableHSTS = true
	cfg.Security.AllowedAPIEndpoints = []string{fakeAnthropic.URL}
	cfg.Security.APIKeyMinLength = 10
	cfg.Anthropic.BaseURL = fakeAnthropic.URL
	cfg.Anthropic.APIVersion = "2023-06-01"
	cfg.Anthropic.KeyPrefix = "sk-ant-"
	cfg.Anthropic.MaxTokens = 100
	cfg.Anthropic.Temperature = 0.7
	cfg.Anthropic.Timeout = config.Duration{Duration: 10 * time.Second}
	cfg.Validation.MaxMessageLength = 1000

	anthropicService := services.NewAnthropicService(cfg)
	apiHandlers := handlers.NewAPIHandlers(cfg, anthropicService)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(security.SecurityHeaders(cfg))
	r.Get("/api/models", apiHandlers.ModelsHandler)
	r.Post("/api/messages", apiHandlers.MessagesHandler)

	server := httptest.NewServer(r)
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	t.Run("happy path: POST /api/messages", func(t *testing.T) {
		reqBody := `{"model":"claude-3-5-haiku","messages":[{"role":"user","content":"hello"}]}`
		req, _ := http.NewRequest("POST", server.URL+"/api/messages", strings.NewReader(reqBody))
		req.Header.Set("x-api-key", "sk-ant-valid123456")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("expected status 200, got %d: %s", resp.StatusCode, body)
		}

		if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if response["role"] != "assistant" {
			t.Error("response should have role=assistant")
		}
		if content, ok := response["content"].([]interface{}); ok {
			if len(content) == 0 {
				t.Error("response should have content")
			}
		}
	})

	t.Run("error mapping: invalid API key", func(t *testing.T) {
		reqBody := `{"model":"claude-3-5-haiku","messages":[{"role":"user","content":"hello"}]}`
		req, _ := http.NewRequest("POST", server.URL+"/api/messages", strings.NewReader(reqBody))
		req.Header.Set("x-api-key", "sk-ant-invalid")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}

		var errorResp map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			t.Fatalf("failed to parse error response: %v", err)
		}

		if !strings.Contains(errorResp["error"], "Invalid API key") {
			t.Errorf("expected 'Invalid API key' error, got: %s", errorResp["error"])
		}
	})

	t.Run("models endpoint with valid key", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/api/models", nil)
		req.Header.Set("x-api-key", "sk-ant-valid123456")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var models map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if data, ok := models["data"].([]interface{}); ok {
			if len(data) != 2 {
				t.Errorf("expected 2 models, got %d", len(data))
			}
		} else {
			t.Error("models response should have 'data' array")
		}
	})
}

func TestSecurityHeadersConditional(t *testing.T) {
	t.Run("HSTS disabled", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Security.EnableHSTS = false
		cfg.Security.AllowedAPIEndpoints = []string{"https://api.anthropic.com"}

		handler := security.SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if hsts := w.Header().Get("Strict-Transport-Security"); hsts != "" {
			t.Error("HSTS header should not be set when EnableHSTS is false")
		}
	})

	t.Run("HSTS enabled", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Security.EnableHSTS = true
		cfg.Security.AllowedAPIEndpoints = []string{"https://api.anthropic.com"}

		handler := security.SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if hsts := w.Header().Get("Strict-Transport-Security"); hsts == "" {
			t.Error("HSTS header should be set when EnableHSTS is true")
		}
	})

	t.Run("CSP includes data: for images", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Security.AllowedAPIEndpoints = []string{"https://api.example.com"}

		handler := security.SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		csp := w.Header().Get("Content-Security-Policy")
		if !strings.Contains(csp, "img-src 'self' data:") {
			t.Errorf("CSP should allow data: URIs for images, got: %s", csp)
		}
		if !strings.Contains(csp, "connect-src 'self' https://api.example.com") {
			t.Errorf("CSP should include allowed API endpoints, got: %s", csp)
		}
	})
}

func TestConfigDefaults(t *testing.T) {
	t.Run("Duration defaults are applied", func(t *testing.T) {
		t.Setenv("MANTO_ENV", "test")
		t.Setenv("MANTO_ANTHROPIC_API_VERSION", "2023-06-01")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		if cfg.Server.ReadTimeout.Duration == 0 {
			t.Error("Server.ReadTimeout default was not applied")
		}
		if cfg.Server.WriteTimeout.Duration == 0 {
			t.Error("Server.WriteTimeout default was not applied")
		}
		if cfg.Anthropic.Timeout.Duration == 0 {
			t.Error("Anthropic.Timeout default was not applied")
		}

		if cfg.Server.ReadTimeout.Duration != 30*time.Second {
			t.Errorf("expected ReadTimeout 30s, got %v", cfg.Server.ReadTimeout.Duration)
		}
		if cfg.Server.WriteTimeout.Duration != 30*time.Second {
			t.Errorf("expected WriteTimeout 30s, got %v", cfg.Server.WriteTimeout.Duration)
		}
		if cfg.Anthropic.Timeout.Duration != 60*time.Second {
			t.Errorf("expected Anthropic.Timeout 60s, got %v", cfg.Anthropic.Timeout.Duration)
		}
	})
}
