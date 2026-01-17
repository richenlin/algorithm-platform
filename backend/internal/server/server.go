package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	v1 "algorithm-platform/api/v1/proto"
	"algorithm-platform/internal/config"
	"algorithm-platform/internal/service"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	grpcServer    *grpc.Server
	httpServer    *http.Server
	httpMux       *http.ServeMux
	mux           *runtime.ServeMux
	managementSvc *service.ManagementService
	cfg           config.ServerConfig
}

func New(cfg config.ServerConfig, managementSvc *service.ManagementService) *Server {
	grpcServer := grpc.NewServer()

	mux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(func(ctx context.Context, w http.ResponseWriter, resp proto.Message) error {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
			return nil
		}),
		runtime.WithErrorHandler(func(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

			runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
		}),
		runtime.WithForwardResponseOption(func(ctx context.Context, w http.ResponseWriter, m proto.Message) error {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			return nil
		}),
	)

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/api/v1/data-download", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("=== data-download called: %s %s\n", r.Method, r.URL.Path)
		fmt.Printf("Query: %v\n", r.URL.Query())
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Expose-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		fileID := r.URL.Query().Get("file_id")
		fmt.Printf("File ID: %s\n", fileID)
		if fileID == "" {
			http.Error(w, "File ID is required", http.StatusBadRequest)
			return
		}

		presignedURL, err := managementSvc.GetPresetDataDownloadURL(r.Context(), fileID)
		if err != nil {
			fmt.Printf("Error generating presigned URL: %v\n", err)
			http.Error(w, fmt.Sprintf("Failed to generate download URL: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"download_url": "%s"}`, presignedURL)
	})
	httpMux.HandleFunc("/api/v1/data/upload-multipart", handleUploadMultipart(managementSvc))
	httpMux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test ok"))
	})
	httpMux.Handle("/api/", corsMiddleware(mux))

	return &Server{
		grpcServer:    grpcServer,
		httpServer:    &http.Server{},
		mux:           mux,
		httpMux:       httpMux,
		managementSvc: managementSvc,
		cfg:           cfg,
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
		s.httpServer.Addr = fmt.Sprintf("0.0.0.0:%d", s.cfg.HTTPPort)
		s.httpServer.Handler = s.httpMux

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

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleUploadMultipart(managementSvc *service.ManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := r.ParseMultipartForm(32 << 20) // 32MB max memory
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse multipart form: %v", err), http.StatusBadRequest)
			return
		}

		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get file: %v", err), http.StatusBadRequest)
			return
		}
		defer file.Close()

		filename := r.FormValue("filename")
		category := r.FormValue("category")

		if filename == "" {
			filename = fileHeader.Filename
		}

		if category == "" {
			category = "通用"
		}

		result, err := managementSvc.UploadPresetDataFile(r.Context(), filename, category, fileHeader.Filename, file)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to upload file: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"file_id": "%s", "minio_url": "%s"}`, result.FileId, result.MinioUrl)
	}
}

func handleDownloadData(managementSvc *service.ManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("=== handleDownloadData called: %s %s ===\n", r.Method, r.URL.Path)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		fileID := r.URL.Query().Get("file_id")
		fmt.Printf("File ID: %s\n", fileID)
		if fileID == "" {
			http.Error(w, "File ID is required", http.StatusBadRequest)
			return
		}

		presignedURL, err := managementSvc.GetPresetDataDownloadURL(r.Context(), fileID)
		if err != nil {
			fmt.Printf("Error generating presigned URL: %v\n", err)
			http.Error(w, fmt.Sprintf("Failed to generate download URL: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"download_url": "%s"}`, presignedURL)
	}
}
