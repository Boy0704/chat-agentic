package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	LLM       LLMConfig       `yaml:"llm"`
	DB        DBConfig        `yaml:"db"`
	ClientAPI ClientAPIConfig `yaml:"client_api"`
	Skills    SkillsConfig    `yaml:"skills"`
	Log       LogConfig       `yaml:"log"`
}

type LogConfig struct {
	Path       string `yaml:"path"`        // kosong = stdout
	Level      string `yaml:"level"`       // debug | info | warn | error
	MaxSizeMB  int    `yaml:"max_size_mb"` // per file, default 100
	MaxAgeDays int    `yaml:"max_age_days"`
	MaxBackups int    `yaml:"max_backups"`
	Compress   bool   `yaml:"compress"`
}

type ServerConfig struct {
	Port         int             `yaml:"port"`
	APIKey       string          `yaml:"api_key"`
	CORS         CORSConfig      `yaml:"cors"`
	RateLimit    RateLimitConfig `yaml:"rate_limit"`
	MaxBodyBytes int64           `yaml:"max_body_bytes"`
}

type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
}

type CORSConfig struct {
	// AllowOrigins daftar origin yang diizinkan, contoh: ["https://app.klien.com"]
	// Default: ["*"] (semua origin diizinkan)
	AllowOrigins []string `yaml:"allow_origins"`
}

type LLMConfig struct {
	BaseURL        string `yaml:"base_url"`
	APIKey         string `yaml:"api_key"`
	Model          string `yaml:"model"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type DBConfig struct {
	Path string `yaml:"path"`
}

// ClientAPIConfig adalah konfigurasi untuk memanggil API sistem klien.
// Skills menggunakan ini untuk fetch data dari sistem yang sudah berjalan.
type ClientAPIConfig struct {
	BaseURL        string `yaml:"base_url"`
	AuthHeader     string `yaml:"auth_header"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type SkillsConfig struct {
	CustomPath     string `yaml:"custom_path"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:         8080,
			MaxBodyBytes: 1 * 1024 * 1024, // 1 MB
			RateLimit:    RateLimitConfig{Enabled: false, RequestsPerMinute: 60},
		},
		LLM:       LLMConfig{TimeoutSeconds: 30},
		ClientAPI: ClientAPIConfig{TimeoutSeconds: 10},
		Skills: SkillsConfig{TimeoutSeconds: 30},
		Log: LogConfig{
			Level:      "info",
			MaxSizeMB:  100,
			MaxAgeDays: 30,
			MaxBackups: 5,
			Compress:   true,
		},
	}

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config: %w", err)
		}
		if err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse config: %w", err)
			}
		}
	}

	overrideFromEnv(cfg)
	return cfg, nil
}

func overrideFromEnv(cfg *Config) {
	if v := os.Getenv("SERVER_PORT"); v != "" {
		cfg.Server.Port, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("SERVER_API_KEY"); v != "" {
		cfg.Server.APIKey = v
	}
	if v := os.Getenv("CORS_ALLOW_ORIGINS"); v != "" {
		cfg.Server.CORS.AllowOrigins = strings.Split(v, ",")
	}
	if v := os.Getenv("LLM_BASE_URL"); v != "" {
		cfg.LLM.BaseURL = v
	}
	if v := os.Getenv("LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.DB.Path = v
	}
	if v := os.Getenv("CLIENT_API_BASE_URL"); v != "" {
		cfg.ClientAPI.BaseURL = v
	}
	if v := os.Getenv("CLIENT_API_AUTH"); v != "" {
		cfg.ClientAPI.AuthHeader = v
	}
	if v := os.Getenv("CUSTOM_SKILLS_PATH"); v != "" {
		cfg.Skills.CustomPath = v
	}
	if v := os.Getenv("LOG_PATH"); v != "" {
		cfg.Log.Path = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
}

func (c *Config) Validate() error {
	if c.Server.APIKey == "" {
		return fmt.Errorf("server.api_key is required")
	}
	if c.LLM.BaseURL == "" {
		return fmt.Errorf("llm.base_url is required")
	}
	if c.LLM.APIKey == "" {
		return fmt.Errorf("llm.api_key is required")
	}
	if c.LLM.Model == "" {
		return fmt.Errorf("llm.model is required")
	}
	if c.DB.Path == "" {
		c.DB.Path = "./data/agent.db"
	}
	return nil
}
