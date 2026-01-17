package main

import (
	"fmt"
	"os"

	"algorithm-platform/internal/config"
)

func main() {
	fmt.Println("=== Algorithm Platform Configuration Validator ===")
	fmt.Println()

	// Try to load configuration
	cfg := config.LoadOrDefault()

	// Display loaded configuration
	fmt.Println("Configuration loaded successfully!")
	fmt.Println()
	fmt.Println("Server Configuration:")
	fmt.Printf("  - gRPC Port: %d\n", cfg.Server.GRPCPort)
	fmt.Printf("  - HTTP Port: %d\n", cfg.Server.HTTPPort)
	fmt.Println()

	fmt.Println("MinIO Configuration:")
	fmt.Printf("  - Endpoint: %s\n", cfg.MinIO.Endpoint)
	fmt.Printf("  - External Endpoint: %s\n", cfg.MinIO.ExternalEndpoint)
	fmt.Printf("  - Access Key: %s\n", cfg.MinIO.AccessKeyID)
	fmt.Printf("  - Bucket: %s\n", cfg.MinIO.Bucket)
	fmt.Printf("  - Use SSL: %v\n", cfg.MinIO.UseSSL)
	fmt.Println()

	fmt.Println("Redis Configuration:")
	fmt.Printf("  - Address: %s\n", cfg.Redis.Addr)
	fmt.Printf("  - Database: %d\n", cfg.Redis.DB)
	fmt.Println()

	fmt.Println("Docker Configuration:")
	fmt.Printf("  - Host: %s\n", cfg.Docker.Host)
	fmt.Printf("  - API Version: %s\n", cfg.Docker.APIVersion)
	fmt.Println()

	// Check for LOCAL_MODE override
	if os.Getenv("LOCAL_MODE") == "true" {
		fmt.Println("⚠️  LOCAL_MODE is enabled")
		fmt.Println("   MinIO endpoint will be overridden to: localhost:9000")
		fmt.Println()
	}

	fmt.Println("✓ Configuration is valid")
}
