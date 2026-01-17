package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	v1 "algorithm-platform/api/v1/proto"
	"algorithm-platform/internal/config"
	"algorithm-platform/internal/database"
	"algorithm-platform/internal/models"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AlgorithmService struct {
	v1.UnimplementedAlgorithmServiceServer
	db          *database.Database
	cfg         *config.Config
	minioClient *minio.Client
}

func NewAlgorithmService(db *database.Database, cfg *config.Config) *AlgorithmService {
	minioClient, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		fmt.Printf("Failed to initialize MinIO client: %v\n", err)
	}
	return &AlgorithmService{
		db:          db,
		cfg:         cfg,
		minioClient: minioClient,
	}
}

func (s *AlgorithmService) ExecuteAlgorithm(ctx context.Context, req *v1.ExecuteRequest) (*v1.ExecuteResponse, error) {
	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())

	if req.IsAsync && req.WebhookUrl == "" {
		return nil, fmt.Errorf("webhook_url is required when is_async is true")
	}

	algorithm := &models.Algorithm{}
	if err := s.db.DB().First(algorithm, "id = ?", req.AlgorithmId).Error; err != nil {
		return nil, fmt.Errorf("algorithm not found: %w", err)
	}

	if _, err := s.checkPlatformConsistency(algorithm.Platform); err != nil {
		return nil, fmt.Errorf("platform consistency check failed: %w", err)
	}

	inputDir := filepath.Join("/tmp", "input", jobID)
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create input directory: %w", err)
	}

	if req.InputSource != nil {
		if err := s.downloadPresetData(ctx, req.InputSource, inputDir); err != nil {
			return nil, fmt.Errorf("failed to download preset data: %w", err)
		}
	}

	if req.Params != nil {
		paramsFile := filepath.Join(inputDir, "params.json")
		paramsJSON := fmt.Sprintf(`%v`, req.Params)
		if err := os.WriteFile(paramsFile, []byte(paramsJSON), 0644); err != nil {
			return nil, fmt.Errorf("failed to write params file: %w", err)
		}
	}

	job := &models.Job{
		ID:            jobID,
		AlgorithmID:   req.AlgorithmId,
		AlgorithmName: algorithm.Name,
		Mode:          req.Mode,
		Status:        "pending",
		InputParams:   fmt.Sprintf("%v", req.Params),
		InputURL:      req.InputSource.GetUrl(),
		WorkerID:      "default-worker",
		CreatedAt:     time.Now(),
	}

	if err := s.db.DB().Create(job).Error; err != nil {
		return nil, fmt.Errorf("failed to create job record: %w", err)
	}

	if req.IsAsync {
		go s.runJobAsync(ctx, jobID, req, algorithm, inputDir)
		return &v1.ExecuteResponse{
			JobId:   jobID,
			Status:  "pending",
			Message: fmt.Sprintf("Async job %s queued for execution", jobID),
		}, nil
	}

	result, err := s.runJobSync(ctx, jobID, req, algorithm, inputDir)
	if err != nil {
		job.Status = "failed"
		job.FinishedAt = &[]time.Time{time.Now()}[0]
		if err := s.db.DB().Save(job).Error; err != nil {
			fmt.Printf("Failed to update job status: %v\n", err)
		}
		return nil, err
	}

	return result, nil
}

func (s *AlgorithmService) GetJobStatus(ctx context.Context, req *v1.GetJobStatusRequest) (*v1.GetJobStatusResponse, error) {
	job := &models.Job{}
	if err := s.db.DB().First(job, "job_id = ?", req.JobId).Error; err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	status := job.Status
	if job.Status == "pending" {
		status = "queued"
	} else if job.Status == "running" {
		status = "running"
	} else if job.Status == "completed" || job.Status == "failed" {
		status = "completed"
	}

	response := &v1.GetJobStatusResponse{
		JobId:      job.ID,
		Status:     status,
		ResultUrl:  job.OutputURL,
		StartedAt:  timestampProto(job.StartedAt),
		FinishedAt: timestampProto(job.FinishedAt),
		CostTimeMs: int32(job.CostTimeMs),
	}

	if job.Status == "pending" {
		response.Status = "queued"
	}

	return response, nil
}

func (s *AlgorithmService) checkPlatformConsistency(algorithmPlatform string) (*v1.GetServerInfoResponse, error) {
	bucketName := "algorithm-platform"

	exists, err := s.minioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check MinIO bucket: %w", err)
	}
	if !exists {
		if err := s.minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create MinIO bucket: %w", err)
		}
	}

	return &v1.GetServerInfoResponse{
		Os:           "linux",
		Arch:         "x86_64",
		Platform:     v1.Platform_PLATFORM_LINUX_X86_64,
		PlatformName: "Linux x86_64",
	}, nil
}

func (s *AlgorithmService) downloadPresetData(ctx context.Context, inputSource *v1.InputSource, targetDir string) error {
	if inputSource.Url == "" {
		return nil
	}

	bucketName := "algorithm-platform"

	presetData := &models.PresetData{}
	if err := s.db.DB().First(presetData, "minio_url = ?", inputSource.Url).Error; err != nil {
		return fmt.Errorf("preset data not found: %w", err)
	}

	minioPath := presetData.MinioURL
	if idx := len(minioPath) - 1; idx > 0 && minioPath[idx] == '/' {
		minioPath = minioPath[:idx]
	}

	bucketName = "algorithm-platform"
	if idx := len(minioPath) - 1; idx >= 0 {
		if minioPath[idx] == '/' {
			bucketName = minioPath[:idx]
			minioPath = ""
		}
	}

	obj, err := s.minioClient.GetObject(ctx, bucketName, minioPath, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get preset data from MinIO: %w", err)
	}
	defer obj.Close()

	filename := filepath.Join(targetDir, filepath.Base(presetData.Filename))
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, obj); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	return nil
}

func (s *AlgorithmService) runJobSync(ctx context.Context, jobID string, req *v1.ExecuteRequest, algorithm *models.Algorithm, inputDir string) (*v1.ExecuteResponse, error) {
	job := &models.Job{}
	s.db.DB().First(job, "job_id = ?", jobID)

	job.Status = "running"
	now := time.Now()
	job.StartedAt = &now
	s.db.DB().Save(job)

	resultURL, err := s.executeInContainer(ctx, jobID, algorithm, inputDir, req.ResourceConfig, req.TimeoutSeconds)

	endTime := time.Now()
	job.FinishedAt = &endTime
	job.CostTimeMs = endTime.Sub(now).Milliseconds()

	if err != nil {
		job.Status = "failed"
		job.LogURL = ""
	} else {
		job.Status = "completed"
		job.OutputURL = resultURL
	}
	s.db.DB().Save(job)

	return &v1.ExecuteResponse{
		JobId:     jobID,
		Status:    job.Status,
		ResultUrl: resultURL,
		Message:   getJobMessage(job.Status, err),
	}, nil
}

func (s *AlgorithmService) runJobAsync(ctx context.Context, jobID string, req *v1.ExecuteRequest, algorithm *models.Algorithm, inputDir string) {
	result, err := s.runJobSync(ctx, jobID, req, algorithm, inputDir)

	if req.WebhookUrl != "" {
		s.sendWebhook(ctx, req.WebhookUrl, jobID, result, err)
	}
}

func (s *AlgorithmService) executeInContainer(ctx context.Context, jobID string, algorithm *models.Algorithm, inputDir string, resourceConfig *v1.ResourceConfig, timeoutSeconds int32) (string, error) {
	return fmt.Sprintf("http://localhost:9000/algorithm-platform/results/%s", jobID), nil
}

func (s *AlgorithmService) sendWebhook(ctx context.Context, webhookURL, jobID string, result *v1.ExecuteResponse, err error) {
	webhookData := map[string]interface{}{
		"job_id":     jobID,
		"status":     result.Status,
		"result_url": result.ResultUrl,
		"message":    result.Message,
		"error":      "",
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	if err != nil {
		webhookData["error"] = err.Error()
		webhookData["status"] = "failed"
	}
}

func getJobMessage(status string, err error) string {
	messages := map[string]string{
		"pending":   "Job is pending",
		"running":   "Job is running",
		"completed": "Job completed successfully",
		"failed":    "Job execution failed",
	}

	if err != nil {
		return fmt.Sprintf("Job failed: %v", err)
	}

	if msg, ok := messages[status]; ok {
		return msg
	}
	return "Job status: " + status
}

func timestampProto(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
