package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"algorithm-platform/api/v1/proto"
	"algorithm-platform/internal/config"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer *grpc.Server
	httpServer *http.Server
	mux        *runtime.ServeMux

	cfg config.ServerConfig
}

func New(cfg config.ServerConfig) *Server {
	grpcServer := grpc.NewServer()
	mux := runtime.NewServeMux()

	return &Server{
		grpcServer: grpcServer,
		mux:        mux,
		cfg:        cfg,
	}
}

func (s *Server) RegisterServices(
	algorithmSvc v1.AlgorithmServiceServer,
	managementSvc v1.ManagementServiceServer,
) {
	v1.RegisterAlgorithmServiceServer(s.grpcServer, algorithmSvc)
	v1.RegisterManagementServiceServer(s.grpcServer, managementSvc)
}

func (s *Server) RegisterGateway(ctx context.Context) error {
	grpcAddr := fmt.Sprintf("0.0.0.0:%d", s.cfg.GRPCPort)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if err := v1.RegisterAlgorithmServiceHandlerFromEndpoint(ctx, s.mux, grpcAddr, opts); err != nil {
		return err
	}

	if err := v1.RegisterManagementServiceHandlerFromEndpoint(ctx, s.mux, grpcAddr, opts); err != nil {
		return err
	}

	return nil
}

func (s *Server) Start(ctx context.Context) error {
	listen, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", s.cfg.GRPCPort))
	if err != nil {
		return err
	}

	reflection.Register(s.grpcServer)

	go func() {
		s.httpServer = &http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:%d", s.cfg.HTTPPort),
			Handler: s.mux,
		}

		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	go func() {
		if err := s.grpcServer.Serve(listen); err != nil {
			panic(err)
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
	}

	s.grpcServer.GracefulStop()
	return nil
}
