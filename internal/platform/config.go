package platform

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig
	DB       DBConfig
	Redis    RedisConfig
	Keycloak KeycloakConfig
	OTP      OTPConfig
	AWS      AWSConfig
}

type ServerConfig struct {
	Host string
	Port int
	Env  string // "development", "staging", "production"
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (c DBConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode)
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type KeycloakConfig struct {
	BaseURL      string
	Realm        string
	AdminUser    string
	AdminPass    string
	ClientID     string
	ClientSecret string
}

type OTPConfig struct {
	Provider  string // "msg91" or "mock"
	AuthKey   string
	TemplateID string
	SendOTPURL string
}

type AWSConfig struct {
	Region   string
	S3Bucket string
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() (*Config, error) {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Server defaults
	v.SetDefault("SERVER_HOST", "0.0.0.0")
	v.SetDefault("SERVER_PORT", 8080)
	v.SetDefault("SERVER_ENV", "development")

	// Database defaults
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_USER", "landintel")
	v.SetDefault("DB_PASSWORD", "landintel")
	v.SetDefault("DB_NAME", "landintel")
	v.SetDefault("DB_SSLMODE", "disable")

	// Redis defaults
	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", 6379)
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 0)

	// Keycloak defaults
	v.SetDefault("KEYCLOAK_BASE_URL", "http://localhost:8180")
	v.SetDefault("KEYCLOAK_REALM", "landintel")
	v.SetDefault("KEYCLOAK_ADMIN_USER", "admin")
	v.SetDefault("KEYCLOAK_ADMIN_PASS", "admin")
	v.SetDefault("KEYCLOAK_CLIENT_ID", "web-app")
	v.SetDefault("KEYCLOAK_CLIENT_SECRET", "")

	// OTP defaults
	v.SetDefault("OTP_PROVIDER", "mock")
	v.SetDefault("OTP_AUTH_KEY", "")
	v.SetDefault("OTP_TEMPLATE_ID", "")
	v.SetDefault("OTP_SEND_URL", "https://api.msg91.com/api/v5/otp")

	// AWS defaults
	v.SetDefault("AWS_REGION", "ap-south-1")
	v.SetDefault("AWS_S3_BUCKET", "landintel-media")

	cfg := &Config{
		Server: ServerConfig{
			Host: v.GetString("SERVER_HOST"),
			Port: v.GetInt("SERVER_PORT"),
			Env:  v.GetString("SERVER_ENV"),
		},
		DB: DBConfig{
			Host:     v.GetString("DB_HOST"),
			Port:     v.GetInt("DB_PORT"),
			User:     v.GetString("DB_USER"),
			Password: v.GetString("DB_PASSWORD"),
			Name:     v.GetString("DB_NAME"),
			SSLMode:  v.GetString("DB_SSLMODE"),
		},
		Redis: RedisConfig{
			Host:     v.GetString("REDIS_HOST"),
			Port:     v.GetInt("REDIS_PORT"),
			Password: v.GetString("REDIS_PASSWORD"),
			DB:       v.GetInt("REDIS_DB"),
		},
		Keycloak: KeycloakConfig{
			BaseURL:      v.GetString("KEYCLOAK_BASE_URL"),
			Realm:        v.GetString("KEYCLOAK_REALM"),
			AdminUser:    v.GetString("KEYCLOAK_ADMIN_USER"),
			AdminPass:    v.GetString("KEYCLOAK_ADMIN_PASS"),
			ClientID:     v.GetString("KEYCLOAK_CLIENT_ID"),
			ClientSecret: v.GetString("KEYCLOAK_CLIENT_SECRET"),
		},
		OTP: OTPConfig{
			Provider:   v.GetString("OTP_PROVIDER"),
			AuthKey:    v.GetString("OTP_AUTH_KEY"),
			TemplateID: v.GetString("OTP_TEMPLATE_ID"),
			SendOTPURL: v.GetString("OTP_SEND_URL"),
		},
		AWS: AWSConfig{
			Region:   v.GetString("AWS_REGION"),
			S3Bucket: v.GetString("AWS_S3_BUCKET"),
		},
	}

	return cfg, nil
}
