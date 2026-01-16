package config

type Config struct {
	Server ServerConfig
	Docker DockerConfig
	Redis  RedisConfig
	MinIO  MinIOConfig
	DB     DBConfig
}

type ServerConfig struct {
	GRPCPort int
	HTTPPort int
}

type DockerConfig struct {
	Host       string
	TLSCert    string
	TLSKey     string
	APIVersion string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type MinIOConfig struct {
	Endpoint         string
	ExternalEndpoint string
	AccessKeyID      string
	SecretAccessKey  string
	Bucket           string
	UseSSL           bool
}

type DBConfig struct {
	DSN string
}

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
	}
}
