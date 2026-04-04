package config

import (
	"time"
)

type Config struct {
	// 既存フィールド（後方互換性維持）
	PortalName string `yaml:"portal_name" env:"PORTAL_NAME" default:"TACOKUMO Portal"`

	// 新規フィールド（段階的追加）
	Server   ServerConfig   `yaml:"server"`
	Auth     AuthConfig     `yaml:"auth"`
	Security SecurityConfig `yaml:"security"`
}

type ServerConfig struct {
	Port     int    `yaml:"port" env:"SERVER_PORT" default:"8080"`
	LogLevel string `yaml:"log_level" env:"LOG_LEVEL" default:"info"`
}

type AuthConfig struct {
	GitHub GitHubConfig `yaml:"github"`
	JWT    JWTConfig    `yaml:"jwt"`
	Valkey ValkeyConfig `yaml:"valkey"`
}

type GitHubConfig struct {
	OAuth GitHubOAuthConfig `yaml:"oauth"`
	App   GitHubAppConfig   `yaml:"app"`
}

type GitHubOAuthConfig struct {
	ClientID     string `yaml:"-" env:"GITHUB_CLIENT_ID"`
	ClientSecret string `yaml:"-" env:"GITHUB_CLIENT_SECRET"`
	RedirectURL  string `yaml:"redirect_url" env:"GITHUB_OAUTH_REDIRECT_URL"`
}

type GitHubAppConfig struct {
	AppID          string `yaml:"-" env:"GITHUB_APP_ID"`
	PrivateKeyPath string `yaml:"-" env:"GITHUB_APP_PRIVATE_KEY_PATH"`
}

type JWTConfig struct {
	PrivateKeyPath       string        `yaml:"-" env:"JWT_PRIVATE_KEY_PATH"`
	PublicKeyPath        string        `yaml:"-" env:"JWT_PUBLIC_KEY_PATH"`
	AccessTokenDuration  time.Duration `yaml:"access_token_duration" env:"JWT_ACCESS_TOKEN_DURATION" default:"1h"`
	RefreshTokenDuration time.Duration `yaml:"refresh_token_duration" env:"JWT_REFRESH_TOKEN_DURATION" default:"8h"`
}

type ValkeyConfig struct {
	Address  string `yaml:"address" env:"VALKEY_ADDRESS" default:"localhost:6379"`
	Password string `yaml:"-" env:"VALKEY_PASSWORD"`
	DB       int    `yaml:"db" env:"VALKEY_DB" default:"0"`
}

type SecurityConfig struct {
	CORS      CORSConfig      `yaml:"cors"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

type RateLimitConfig struct {
	// IP rate limiting: requests per period
	IPRate  int `yaml:"ip_rate" env:"RATE_LIMIT_IP_RATE" default:"60"`
	IPBurst int `yaml:"ip_burst" env:"RATE_LIMIT_IP_BURST" default:"20"`
	// User rate limiting
	UserRate  int `yaml:"user_rate" env:"RATE_LIMIT_USER_RATE" default:"300"`
	UserBurst int `yaml:"user_burst" env:"RATE_LIMIT_USER_BURST" default:"60"`
	// Period in seconds (shared by IP and user limits)
	PeriodSeconds int `yaml:"period_seconds" env:"RATE_LIMIT_PERIOD_SECONDS" default:"60"`
	// Behavior on Valkey failure: true = allow request, false = reject with 503
	FailOpen bool `yaml:"fail_open" env:"RATE_LIMIT_FAIL_OPEN" default:"true"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins" env:"CORS_ALLOWED_ORIGINS" env-separator:","`
}

// Validateは validator.goに移動するため、ここでは一時的な実装を保持
// 実際の検証ロジックは validator.go で実装される
