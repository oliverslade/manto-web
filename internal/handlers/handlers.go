package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/manto/manto-web/internal/config"
	"github.com/manto/manto-web/internal/services"
)

type APIHandlers struct {
	config           *config.Config
	anthropicService *services.AnthropicService
}

func NewAPIHandlers(cfg *config.Config, anthropicService *services.AnthropicService) *APIHandlers {
	return &APIHandlers{
		config:           cfg,
		anthropicService: anthropicService,
	}
}

func (h *APIHandlers) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	configData := map[string]interface{}{
		"providers": []map[string]string{
			{
				"name":        "anthropic",
				"displayName": "Anthropic",
			},
		},
		"api": map[string]interface{}{
			"anthropicKeyPrefix": h.config.Anthropic.KeyPrefix,
		},
		"validation": map[string]interface{}{
			"maxMessageLength": h.config.Validation.MaxMessageLength,
			"minApiKeyLength":  h.config.Security.APIKeyMinLength,
		},
		"version": "2.0.0",
	}

	jsonData, err := json.Marshal(configData)
	if err != nil {
		http.Error(w, "Failed to generate config", http.StatusInternalServerError)
		return
	}

	configScript := fmt.Sprintf("window.MantoConfig = %s;", string(jsonData))

	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "public, max-age=300") // 5 minutes
	w.Write([]byte(configScript))
}

func (h *APIHandlers) ModelsHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("x-api-key")
	if !h.anthropicService.ValidateAPIKey(apiKey) {
		writeJSONError(w, http.StatusBadRequest, "Invalid API key format", "")
		return
	}

	modelsData, err := h.anthropicService.GetModels(apiKey)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(modelsData))
}

func (h *APIHandlers) MessagesHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("x-api-key")
	if !h.anthropicService.ValidateAPIKey(apiKey) {
		writeJSONError(w, http.StatusBadRequest, "Invalid API key format", "")
		return
	}

	var messageRequest services.MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&messageRequest); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON format", "")
		return
	}

	if messageRequest.Model == "" {
		writeJSONError(w, http.StatusBadRequest, "Model is required", "")
		return
	}

	if len(messageRequest.Messages) == 0 {
		writeJSONError(w, http.StatusBadRequest, "Messages are required", "")
		return
	}

	maxLength := h.config.Validation.MaxMessageLength
	for _, msg := range messageRequest.Messages {
		if len(msg.Content) > maxLength {
			writeJSONError(w, http.StatusBadRequest,
				fmt.Sprintf("Message too long (max %d characters)", maxLength), "")
			return
		}
	}

	messageRequest.MaxTokens = h.config.Anthropic.MaxTokens
	messageRequest.Temperature = &h.config.Anthropic.Temperature
	messageRequest.System = &h.config.Anthropic.SystemMessage

	response, err := h.anthropicService.SendMessage(apiKey, &messageRequest)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := map[string]string{
		"error": message,
	}
	if details != "" {
		errorResp["details"] = details
	}

	json.NewEncoder(w).Encode(errorResp)
}
