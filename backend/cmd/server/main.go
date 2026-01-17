package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"algorithm-platform/internal/config"
	"algorithm-platform/internal/database"
	"algorithm-platform/internal/server"
	"algorithm-platform/internal/service"
)

func main() {
	// Load configuration from config.yaml or use default
	cfg := config.LoadOrDefault()

	// Override MinIO endpoint if LOCAL_MODE is set
	localMode := os.Getenv("LOCAL_MODE") == "true"
	if localMode {
		cfg.MinIO.Endpoint = "localhost:9000"
		cfg.MinIO.ExternalEndpoint = "localhost:9000"
		log.Println("LOCAL_MODE enabled: using localhost:9000 for MinIO")
	}

	// Initialize database
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize services
	managementSvc := service.NewManagementService(cfg)
	algorithmSvc := service.NewAlgorithmService(db)
	srv := server.New(cfg.Server, managementSvc)

	srv.RegisterServices(algorithmSvc, managementSvc)

	if err := srv.RegisterGateway(context.Background()); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}

	if err := srv.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Printf("Server started. gRPC: %d, HTTP: %d", cfg.Server.GRPCPort, cfg.Server.HTTPPort)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := srv.Stop(context.Background()); err != nil {
		log.Fatalf("Failed to stop server: %v", err)
	}

	log.Println("Server stopped")
}
