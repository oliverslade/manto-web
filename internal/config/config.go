package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Duration struct {
	time.Duration
}

type Config struct {
	Server     ServerConfig
	Security   SecurityConfig
	Logging    LoggingConfig
	Anthropic  AnthropicConfig
	Validation ValidationConfig
}

type ServerConfig struct {
	Port         int      `env:"PORT" default:"8080"`
	Host         string   `env:"HOST" default:"0.0.0.0"`
	ReadTimeout  Duration `env:"READ_TIMEOUT" default:"30s"`
	WriteTimeout Duration `env:"WRITE_TIMEOUT" default:"30s"`
	AllowedHosts []string `env:"ALLOWED_HOSTS" default:"*"`
}

type SecurityConfig struct {
	EnableHSTS          bool     `env:"ENABLE_HSTS" default:"true"`
	AllowedAPIEndpoints []string `env:"ALLOWED_API_ENDPOINTS" default:"https://api.anthropic.com"`
	APIKeyMinLength     int      `env:"API_KEY_MIN_LENGTH" default:"10"`
}

type LoggingConfig struct {
	Level            string `env:"LOG_LEVEL" default:"info"`
	Format           string `env:"LOG_FORMAT" default:"json"`
	IncludeTimestamp bool   `env:"LOG_INCLUDE_TIMESTAMP" default:"true"`
	IncludeSource    bool   `env:"LOG_INCLUDE_SOURCE" default:"false"`
}

type AnthropicConfig struct {
	APIKey        string   `env:"ANTHROPIC_API_KEY"`
	BaseURL       string   `env:"ANTHROPIC_BASE_URL" default:"https://api.anthropic.com"`
	APIVersion    string   `env:"ANTHROPIC_API_VERSION" default:"2023-06-01"`
	Timeout       Duration `env:"ANTHROPIC_TIMEOUT" default:"60s"`
	MaxRetries    int      `env:"ANTHROPIC_MAX_RETRIES" default:"3"`
	KeyPrefix     string   `env:"ANTHROPIC_KEY_PREFIX" default:"sk-ant-"`
	DefaultModel  string   `env:"ANTHROPIC_DEFAULT_MODEL" default:"claude-3-5-haiku"`
	MaxTokens     int      `env:"ANTHROPIC_MAX_TOKENS" default:"1024"`
	Temperature   float64  `env:"ANTHROPIC_TEMPERATURE" default:"0.7"`
	SystemMessage string   `env:"ANTHROPIC_SYSTEM_MESSAGE" default:"Be concise in your responses unless asked otherwise. Prefer tables and short paragraphs."`
}

type ValidationConfig struct {
	MaxMessageLength int `env:"MAX_MESSAGE_LENGTH" default:"4000"`
	MaxFileSize      int `env:"MAX_FILE_SIZE" default:"10485760"` // 10MB
}

func Load() (*Config, error) {
	cfg := &Config{}

	loadEnvFiles()

	if err := setDefaults(cfg); err != nil {
		return nil, fmt.Errorf("failed to set defaults: %w", err)
	}

	if err := loadFromEnv(cfg); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

func loadEnvFiles() {
	env := GetEnvironment()

	envFiles := []string{
		fmt.Sprintf(".env.%s.local", env),
		fmt.Sprintf(".env.%s", env),
		".env.local",
		".env",
	}

	for _, file := range envFiles {
		if _, err := os.Stat(file); err == nil {
			_ = godotenv.Load(file)
		}
	}
}

func GetEnvironment() string {
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}
	if env == "" {
		env = "production"
	}
	return strings.ToLower(env)
}

func loadFromEnv(cfg *Config) error {
	return loadEnvVars(reflect.ValueOf(cfg).Elem(), reflect.TypeOf(cfg).Elem())
}

func loadEnvVars(v reflect.Value, t reflect.Type) error {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.CanSet() {
			continue
		}

		if field.Kind() == reflect.Struct && fieldType.Type != reflect.TypeOf(Duration{}) {
			if err := loadEnvVars(field, fieldType.Type); err != nil {
				return err
			}
			continue
		}

		envTag := fieldType.Tag.Get("env")
		if envTag == "" {
			continue
		}

		envValue := os.Getenv(envTag)
		if envValue == "" {
			continue
		}

		if err := setFieldFromString(field, envValue); err != nil {
			return fmt.Errorf("failed to set field %s from env var %s: %w", fieldType.Name, envTag, err)
		}
	}
	return nil
}

func setDefaults(cfg *Config) error {
	return setDefaultValues(reflect.ValueOf(cfg).Elem(), reflect.TypeOf(cfg).Elem())
}

func setDefaultValues(v reflect.Value, t reflect.Type) error {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.CanSet() {
			continue
		}

		if field.Kind() == reflect.Struct && fieldType.Type.String() != "time.Duration" {
			if err := setDefaultValues(field, fieldType.Type); err != nil {
				return err
			}
			continue
		}

		defaultTag := fieldType.Tag.Get("default")
		if defaultTag == "" {
			continue
		}

		if err := setFieldFromString(field, defaultTag); err != nil {
			return fmt.Errorf("failed to set default for field %s: %w", fieldType.Name, err)
		}
	}
	return nil
}

func setFieldFromString(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intValue)

	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatValue)

	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolValue)

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			var slice []string
			if value != "" {
				slice = strings.Split(value, ",")
				for i, v := range slice {
					slice[i] = strings.TrimSpace(v)
				}
			}
			field.Set(reflect.ValueOf(slice))
		}

	case reflect.Struct:
		if field.Type() == reflect.TypeOf(Duration{}) {
			duration, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(Duration{duration}))
		}

	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

func validate(cfg *Config) error {
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d (must be between 1 and 65535)", cfg.Server.Port)
	}

	if cfg.Security.APIKeyMinLength < 1 {
		return fmt.Errorf("invalid API key minimum length: %d (must be at least 1)", cfg.Security.APIKeyMinLength)
	}

	if cfg.Validation.MaxMessageLength < 1 {
		return fmt.Errorf("invalid max message length: %d (must be at least 1)", cfg.Validation.MaxMessageLength)
	}

	if cfg.Anthropic.MaxTokens < 1 {
		return fmt.Errorf("invalid max tokens: %d (must be at least 1)", cfg.Anthropic.MaxTokens)
	}

	if cfg.Anthropic.Temperature < 0 || cfg.Anthropic.Temperature > 2 {
		return fmt.Errorf("invalid temperature: %f (must be between 0 and 2)", cfg.Anthropic.Temperature)
	}

	validLogLevels := []string{"debug", "info", "warn", "error"}
	found := false
	for _, level := range validLogLevels {
		if cfg.Logging.Level == level {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid log level: %s (must be one of: %s)", cfg.Logging.Level, strings.Join(validLogLevels, ", "))
	}

	return nil
}
