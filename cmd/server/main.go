package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"algorithm-platform/internal/config"
	"algorithm-platform/internal/server"
	"algorithm-platform/internal/service"
)

func main() {
	cfg := config.Default()

	srv := server.New(cfg.Server)

	algorithmSvc := service.NewAlgorithmService()
	managementSvc := service.NewManagementService()

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
