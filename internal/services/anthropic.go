package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/manto/manto-web/internal/config"
)

type AnthropicService struct {
	config     *config.Config
	httpClient *http.Client
}

func NewAnthropicService(cfg *config.Config) *AnthropicService {
	return &AnthropicService{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Anthropic.Timeout.Duration,
		},
	}
}

func (s *AnthropicService) GetModels(apiKey string) (string, error) {
	req, err := http.NewRequest("GET", s.config.Anthropic.BaseURL+"/v1/models", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	s.setHeaders(req, apiKey, s.config.Anthropic.APIVersion)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

func (s *AnthropicService) SendMessage(apiKey string, request *MessageRequest) (*MessageResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.config.Anthropic.BaseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	s.setHeaders(req, apiKey, s.config.Anthropic.APIVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return nil, fmt.Errorf("%s", errorResp.Error.Message)
		}

		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("invalid API key")
		case http.StatusBadRequest:
			return nil, fmt.Errorf("invalid request format")
		case http.StatusTooManyRequests:
			return nil, fmt.Errorf("rate limit exceeded")
		case http.StatusInternalServerError:
			return nil, fmt.Errorf("service temporarily unavailable")
		default:
			return nil, fmt.Errorf("failed to send message")
		}
	}

	var messageResp MessageResponse
	if err := json.Unmarshal(body, &messageResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &messageResp, nil
}

func (s *AnthropicService) ValidateAPIKey(apiKey string) bool {
	prefix := s.config.Anthropic.KeyPrefix
	minLength := s.config.Security.APIKeyMinLength

	return len(apiKey) >= minLength && strings.HasPrefix(apiKey, prefix)
}

func (s *AnthropicService) setHeaders(req *http.Request, apiKey string, apiVersion string) {
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", apiVersion)
	req.Header.Set("User-Agent", "Manto/1.0")
}
