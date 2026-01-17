package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Docker   DockerConfig   `yaml:"docker"`
	Redis    RedisConfig    `yaml:"redis"`
	MinIO    MinIOConfig    `yaml:"minio"`
	Database DatabaseConfig `yaml:"database"`
}

type ServerConfig struct {
	GRPCPort int `yaml:"grpc_port"`
	HTTPPort int `yaml:"http_port"`
}

type DockerConfig struct {
	Host       string `yaml:"host"`
	TLSCert    string `yaml:"tls_cert"`
	TLSKey     string `yaml:"tls_key"`
	APIVersion string `yaml:"api_version"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type MinIOConfig struct {
	Endpoint         string `yaml:"endpoint"`
	ExternalEndpoint string `yaml:"external_endpoint"`
	AccessKeyID      string `yaml:"access_key_id"`
	SecretAccessKey  string `yaml:"secret_access_key"`
	Bucket           string `yaml:"bucket"`
	UseSSL           bool   `yaml:"use_ssl"`
}

type DatabaseConfig struct {
	Type string `yaml:"type"` // sqlite, postgres
	// SQLite 配置
	SQLitePath string `yaml:"sqlite_path"`
	// PostgreSQL 配置
	PostgreSQL PostgreSQLConfig `yaml:"postgresql"`
}

type PostgreSQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"` // disable, require, verify-ca, verify-full
	Timezone string `yaml:"timezone"`
}

// Load loads configuration from config.yaml file
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// LoadOrDefault loads configuration from config.yaml, falls back to default if file not found
func LoadOrDefault() *Config {
	configPaths := []string{
		"config/config.yaml",
		"./config.yaml",
		"../config/config.yaml",
	}

	for _, path := range configPaths {
		if cfg, err := Load(path); err == nil {
			fmt.Printf("Loaded configuration from: %s\n", path)
			return cfg
		}
	}

	fmt.Println("Config file not found, using default configuration")
	return Default()
}

// Default returns default configuration
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			GRPCPort: 9090,
			HTTPPort: 8080,
		},
		Docker: DockerConfig{
			Host:       "unix:///var/run/docker.sock",
			APIVersion: "1.45",
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
			DB:   0,
		},
		MinIO: MinIOConfig{
			Endpoint:         "minio:9000",
			ExternalEndpoint: "localhost:9000",
			AccessKeyID:      "minioadmin",
			SecretAccessKey:  "minioadmin",
			Bucket:           "algorithm-platform",
			UseSSL:           false,
		},
		Database: DatabaseConfig{
			Type:       "sqlite",
			SQLitePath: "./data/algorithm-platform.db",
			PostgreSQL: PostgreSQLConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "postgres",
				DBName:   "algorithm_platform",
				SSLMode:  "disable",
				Timezone: "Asia/Shanghai",
			},
		},
	}
}

// GetConfigPath returns the absolute path to config.yaml
func GetConfigPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(cwd, "config", "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	configPath = filepath.Join(cwd, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	return "", fmt.Errorf("config.yaml not found")
}
