package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"time"

	"algorithm-platform/internal/config"

	v1 "algorithm-platform/api/v1/proto"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ManagementService struct {
	v1.UnimplementedManagementServiceServer

	mu          sync.RWMutex
	algorithms  map[string]*v1.Algorithm
	versions    map[string][]*v1.Version
	presetData  map[string]*v1.PresetData
	minioClient *minio.Client
	bucketName  string
	cfg         *config.Config
}

func NewManagementService(cfg *config.Config) *ManagementService {
	minioClient, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		fmt.Printf("Failed to initialize MinIO client: %v\n", err)
	}

	bucketName := cfg.MinIO.Bucket
	ctx := context.Background()
	if minioClient != nil {
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
			if errBucketExists == nil && exists {
			} else {
				fmt.Printf("Failed to create bucket: %v\n", err)
			}
		}
	}

	return &ManagementService{
		algorithms:  make(map[string]*v1.Algorithm),
		versions:    make(map[string][]*v1.Version),
		presetData:  make(map[string]*v1.PresetData),
		minioClient: minioClient,
		bucketName:  bucketName,
		cfg:         cfg,
	}
}

func (s *ManagementService) CreateAlgorithm(ctx context.Context, req *v1.CreateAlgorithmRequest) (*v1.Algorithm, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("alg_%d", time.Now().UnixNano())

	algorithm := &v1.Algorithm{
		Id:           id,
		Name:         req.Name,
		Description:  req.Description,
		Language:     req.Language,
		Platform:     req.Platform,
		Category:     "",
		Entrypoint:   req.Entrypoint,
		Tags:         req.Tags,
		PresetDataId: req.PresetDataId,
		CreatedAt:    timestamppb.Now(),
		UpdatedAt:    timestamppb.Now(),
	}

	s.algorithms[id] = algorithm

	if len(req.FileData) > 0 && req.FileName != "" {
		minioPath := fmt.Sprintf("algorithms/%s/v1/%s", id, req.FileName)
		if s.minioClient != nil {
			_, err := s.minioClient.PutObject(ctx, s.bucketName, minioPath, bytes.NewReader(req.FileData), int64(len(req.FileData)), minio.PutObjectOptions{
				ContentType: "application/zip",
			})
			if err != nil {
				fmt.Printf("Failed to upload file to MinIO: %v\n", err)
			}
		}

		version := &v1.Version{
			Id:             fmt.Sprintf("ver_%d", time.Now().UnixNano()),
			AlgorithmId:    id,
			VersionNumber:  1,
			MinioPath:      minioPath,
			SourceCodeFile: req.FileName,
			CommitMessage:  "Initial version",
			CreatedAt:      timestamppb.Now(),
		}

		s.versions[id] = append(s.versions[id], version)
		algorithm.CurrentVersionId = version.Id
	}

	return algorithm, nil
}

func (s *ManagementService) UpdateAlgorithm(ctx context.Context, req *v1.UpdateAlgorithmRequest) (*v1.Algorithm, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	algorithm, exists := s.algorithms[req.Id]
	if !exists {
		return nil, fmt.Errorf("algorithm not found")
	}

	algorithm.Name = req.Name
	algorithm.Description = req.Description
	algorithm.Tags = req.Tags
	algorithm.UpdatedAt = timestamppb.Now()

	return algorithm, nil
}

func (s *ManagementService) ListAlgorithms(ctx context.Context, req *v1.ListAlgorithmsRequest) (*v1.ListAlgorithmsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	algorithms := make([]*v1.Algorithm, 0, len(s.algorithms))
	for _, alg := range s.algorithms {
		algorithmCopy := &v1.Algorithm{
			Id:               alg.Id,
			Name:             alg.Name,
			Description:      alg.Description,
			Language:         alg.Language,
			Platform:         alg.Platform,
			Entrypoint:       alg.Entrypoint,
			Tags:             alg.Tags,
			CurrentVersionId: alg.CurrentVersionId,
			CreatedAt:        alg.CreatedAt,
			UpdatedAt:        alg.UpdatedAt,
		}

		algorithms = append(algorithms, algorithmCopy)
	}

	return &v1.ListAlgorithmsResponse{
		Algorithms: algorithms,
		Total:      int32(len(algorithms)),
	}, nil
}

func (s *ManagementService) GetAlgorithm(ctx context.Context, req *v1.GetAlgorithmRequest) (*v1.GetAlgorithmResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alg, exists := s.algorithms[req.Id]
	if !exists {
		return nil, fmt.Errorf("algorithm not found")
	}

	versions := s.versions[req.Id]
	versionsCopy := make([]*v1.Version, len(versions))
	copy(versionsCopy, versions)

	return &v1.GetAlgorithmResponse{
		Algorithm: alg,
		Versions:  versionsCopy,
	}, nil
}

func (s *ManagementService) CreateVersion(ctx context.Context, req *v1.CreateVersionRequest) (*v1.Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	algorithm, exists := s.algorithms[req.AlgorithmId]
	if !exists {
		return nil, fmt.Errorf("algorithm not found")
	}

	existingVersions := s.versions[req.AlgorithmId]
	nextVersionNumber := int32(1)
	if len(existingVersions) > 0 {
		nextVersionNumber = existingVersions[len(existingVersions)-1].VersionNumber + 1
	}

	minioPath := req.SourceCodeZipUrl
	if len(req.FileData) > 0 && req.FileName != "" {
		minioPath = fmt.Sprintf("algorithms/%s/v%d/%s", req.AlgorithmId, nextVersionNumber, req.FileName)
		if s.minioClient != nil {
			_, err := s.minioClient.PutObject(ctx, s.bucketName, minioPath, bytes.NewReader(req.FileData), int64(len(req.FileData)), minio.PutObjectOptions{
				ContentType: "application/zip",
			})
			if err != nil {
				fmt.Printf("Failed to upload file to MinIO: %v\n", err)
				return nil, fmt.Errorf("failed to upload file: %v", err)
			}
		}
	}

	version := &v1.Version{
		Id:             fmt.Sprintf("ver_%d", time.Now().UnixNano()),
		AlgorithmId:    req.AlgorithmId,
		VersionNumber:  nextVersionNumber,
		MinioPath:      minioPath,
		SourceCodeFile: req.FileName,
		CommitMessage:  req.CommitMessage,
		CreatedAt:      timestamppb.Now(),
	}

	s.versions[req.AlgorithmId] = append(s.versions[req.AlgorithmId], version)
	algorithm.CurrentVersionId = version.Id

	return version, nil
}

func (s *ManagementService) RollbackVersion(ctx context.Context, req *v1.RollbackVersionRequest) (*v1.Algorithm, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	algorithm, exists := s.algorithms[req.AlgorithmId]
	if !exists {
		return nil, fmt.Errorf("algorithm not found")
	}

	var targetVersion *v1.Version
	found := false
	for _, version := range s.versions[req.AlgorithmId] {
		if version.Id == req.VersionId {
			targetVersion = version
			found = true
			break
		}
	}

	if !found || targetVersion == nil {
		return nil, fmt.Errorf("version not found")
	}

	algorithm.CurrentVersionId = req.VersionId
	algorithm.UpdatedAt = timestamppb.Now()

	return algorithm, nil
}

func (s *ManagementService) UploadPresetData(ctx context.Context, req *v1.UploadDataRequest) (*v1.UploadDataResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("data_%d", time.Now().UnixNano())

	var minioPath string

	if len(req.FileData) > 0 && req.Filename != "" {
		minioPath = fmt.Sprintf("preset-data/%s", req.Filename)
		if s.minioClient != nil {
			_, err := s.minioClient.PutObject(ctx, s.bucketName, minioPath, bytes.NewReader(req.FileData), int64(len(req.FileData)), minio.PutObjectOptions{})
			if err != nil {
				fmt.Printf("Failed to upload preset data to MinIO: %v\n", err)
				return nil, fmt.Errorf("failed to upload file: %v", err)
			}
		}
	} else if req.MinioPath != "" {
		minioPath = req.MinioPath
	}

	if minioPath == "" {
		return nil, fmt.Errorf("either file_data or minio_path must be provided")
	}

	minioURL := fmt.Sprintf("http://localhost:9000/%s/%s", s.bucketName, minioPath)

	presetData := &v1.PresetData{
		Id:        id,
		Filename:  req.Filename,
		Category:  req.Category,
		MinioUrl:  minioURL,
		CreatedAt: timestamppb.Now(),
	}

	s.presetData[id] = presetData

	return &v1.UploadDataResponse{
		FileId:   id,
		MinioUrl: minioURL,
	}, nil
}

func (s *ManagementService) ListPresetData(ctx context.Context, req *v1.ListPresetDataRequest) (*v1.ListPresetDataResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scheme := "http"
	if s.cfg.MinIO.UseSSL {
		scheme = "https"
	}

	files := make([]*v1.PresetData, 0, len(s.presetData))
	for _, data := range s.presetData {
		if req.Category != "" && data.Category != req.Category {
			continue
		}

		dataCopy := &v1.PresetData{
			Id:        data.Id,
			Filename:  data.Filename,
			Category:  data.Category,
			MinioUrl:  fmt.Sprintf("%s://%s/%s/%s", scheme, s.cfg.MinIO.ExternalEndpoint, s.bucketName, data.MinioUrl),
			CreatedAt: data.CreatedAt,
		}

		files = append(files, dataCopy)
	}

	return &v1.ListPresetDataResponse{
		Files: files,
		Total: int32(len(files)),
	}, nil
}

func (s *ManagementService) GetPresetDataDownloadURL(ctx context.Context, fileID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.presetData[fileID]
	if !exists {
		return "", fmt.Errorf("file not found")
	}

	if s.minioClient == nil {
		return "", fmt.Errorf("minio client not available")
	}

	minioPath := data.MinioUrl
	presignedURL, err := s.minioClient.PresignedGetObject(ctx, s.bucketName, minioPath, time.Hour*24, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %v", err)
	}

	return presignedURL.String(), nil
}

func (s *ManagementService) UploadPresetDataFile(ctx context.Context, filename string, category string, originalFilename string, file io.Reader) (*v1.UploadDataResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("data_%d", time.Now().UnixNano())

	minioPath := fmt.Sprintf("preset-data/%s", originalFilename)
	if s.minioClient != nil {
		_, err := s.minioClient.PutObject(ctx, s.bucketName, minioPath, file, -1, minio.PutObjectOptions{})
		if err != nil {
			fmt.Printf("Failed to upload preset data to MinIO: %v\n", err)
			return nil, fmt.Errorf("failed to upload file: %v", err)
		}
	}

	scheme := "http"
	if s.cfg.MinIO.UseSSL {
		scheme = "https"
	}
	minioURL := fmt.Sprintf("%s://%s/%s/%s", scheme, s.cfg.MinIO.ExternalEndpoint, s.bucketName, minioPath)

	presetData := &v1.PresetData{
		Id:        id,
		Filename:  filename,
		Category:  category,
		MinioUrl:  minioPath,
		CreatedAt: timestamppb.Now(),
	}

	s.presetData[id] = presetData

	return &v1.UploadDataResponse{
		FileId:   id,
		MinioUrl: minioURL,
	}, nil
}

func (s *ManagementService) ListJobs(ctx context.Context, req *v1.ListJobsRequest) (*v1.ListJobsResponse, error) {
	return &v1.ListJobsResponse{
		Jobs:  []*v1.JobSummary{},
		Total: 0,
	}, nil
}

func (s *ManagementService) GetJobDetail(ctx context.Context, req *v1.GetJobDetailRequest) (*v1.JobDetail, error) {
	return &v1.JobDetail{
		JobId:       req.JobId,
		AlgorithmId: "alg_001",
		Mode:        "async",
		Status:      "pending",
	}, nil
}

func (s *ManagementService) GetServerInfo(ctx context.Context, req *v1.GetServerInfoRequest) (*v1.GetServerInfoResponse, error) {
	os := runtime.GOOS
	arch := runtime.GOARCH

	var platform v1.Platform
	var platformName string

	switch {
	case os == "darwin" && arch == "arm64":
		platform = v1.Platform_PLATFORM_MACOS_ARM64
		platformName = "macOS ARM64"
	case os == "windows" && (arch == "amd64" || arch == "386"):
		platform = v1.Platform_PLATFORM_WINDOWS_X86_64
		platformName = "Windows x86_64"
	case os == "linux" && (arch == "amd64" || arch == "386"):
		platform = v1.Platform_PLATFORM_LINUX_X86_64
		platformName = "Linux x86_64"
	case os == "linux" && arch == "arm64":
		platform = v1.Platform_PLATFORM_LINUX_ARM64
		platformName = "Linux ARM64"
	default:
		platform = v1.Platform_PLATFORM_DOCKER
		platformName = fmt.Sprintf("%s %s", strings.Title(os), arch)
	}

	return &v1.GetServerInfoResponse{
		Os:           os,
		Arch:         arch,
		Platform:     platform,
		PlatformName: platformName,
	}, nil
}
