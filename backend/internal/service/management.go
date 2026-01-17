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
	"algorithm-platform/internal/database"
	"algorithm-platform/internal/models"

	v1 "algorithm-platform/api/v1/proto"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ManagementService struct {
	v1.UnimplementedManagementServiceServer

	mu          sync.RWMutex
	db          *database.Database
	minioClient *minio.Client
	bucketName  string
	cfg         *config.Config
}

func NewManagementService(db *database.Database, cfg *config.Config) *ManagementService {
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
		db:          db,
		minioClient: minioClient,
		bucketName:  bucketName,
		cfg:         cfg,
	}
}

// modelToProto 将数据库模型转换为proto格式
func modelToProto(dbAlg *models.Algorithm) *v1.Algorithm {
	tags := []string{}
	if dbAlg.Tags != "" {
		tags = strings.Split(dbAlg.Tags, ",")
	}

	return &v1.Algorithm{
		Id:               dbAlg.ID,
		Name:             dbAlg.Name,
		Description:      dbAlg.Description,
		Language:         dbAlg.Language,
		Platform:         v1.Platform(v1.Platform_value["PLATFORM_"+strings.ToUpper(dbAlg.Platform)]),
		Category:         dbAlg.Category,
		Entrypoint:       dbAlg.Entrypoint,
		Tags:             tags,
		PresetDataId:     dbAlg.PresetDataID,
		CurrentVersionId: dbAlg.CurrentVersionID,
		CreatedAt:        timestamppb.New(dbAlg.CreatedAt),
		UpdatedAt:        timestamppb.New(dbAlg.UpdatedAt),
	}
}

// versionModelToProto 将版本模型转换为proto格式
func versionModelToProto(dbVer *models.Version) *v1.Version {
	return &v1.Version{
		Id:             dbVer.ID,
		AlgorithmId:    dbVer.AlgorithmID,
		VersionNumber:  int32(dbVer.VersionNumber),
		MinioPath:      dbVer.MinioPath,
		SourceCodeFile: dbVer.SourceCodeFile,
		CommitMessage:  dbVer.CommitMessage,
		CreatedAt:      timestamppb.New(dbVer.CreatedAt),
	}
}

// presetDataModelToProto 将预设数据模型转换为proto格式
func presetDataModelToProto(dbData *models.PresetData, scheme, endpoint, bucket string) *v1.PresetData {
	return &v1.PresetData{
		Id:        dbData.ID,
		Filename:  dbData.Filename,
		Category:  dbData.Category,
		MinioUrl:  fmt.Sprintf("%s://%s/%s/%s", scheme, endpoint, bucket, dbData.MinioPath),
		CreatedAt: timestamppb.New(dbData.CreatedAt),
	}
}

func (s *ManagementService) CreateAlgorithm(ctx context.Context, req *v1.CreateAlgorithmRequest) (*v1.Algorithm, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("alg_%d", time.Now().UnixNano())
	now := time.Now()

	// 创建数据库模型
	dbAlgorithm := &models.Algorithm{
		ID:           id,
		Name:         req.Name,
		Description:  req.Description,
		Language:     req.Language,
		Platform:     strings.ToLower(req.Platform.String()),
		Category:     "",
		Entrypoint:   req.Entrypoint,
		Tags:         strings.Join(req.Tags, ","),
		PresetDataID: req.PresetDataId,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// 保存到数据库
	if err := s.db.DB().Create(dbAlgorithm).Error; err != nil {
		return nil, fmt.Errorf("failed to create algorithm: %w", err)
	}

	// 处理文件上传
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

		// 创建版本记录
		dbVersion := &models.Version{
			ID:             fmt.Sprintf("ver_%d", time.Now().UnixNano()),
			AlgorithmID:    id,
			VersionNumber:  1,
			MinioPath:      minioPath,
			SourceCodeFile: req.FileName,
			CommitMessage:  "Initial version",
			CreatedAt:      now,
		}

		if err := s.db.DB().Create(dbVersion).Error; err != nil {
			fmt.Printf("Failed to create version: %v\n", err)
		} else {
			// 更新算法的当前版本ID
			dbAlgorithm.CurrentVersionID = dbVersion.ID
			s.db.DB().Save(dbAlgorithm)
		}
	}

	return modelToProto(dbAlgorithm), nil
}

func (s *ManagementService) UpdateAlgorithm(ctx context.Context, req *v1.UpdateAlgorithmRequest) (*v1.Algorithm, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var dbAlgorithm models.Algorithm
	if err := s.db.DB().First(&dbAlgorithm, "id = ?", req.Id).Error; err != nil {
		return nil, fmt.Errorf("algorithm not found: %w", err)
	}

	dbAlgorithm.Name = req.Name
	dbAlgorithm.Description = req.Description
	dbAlgorithm.Tags = strings.Join(req.Tags, ",")
	dbAlgorithm.UpdatedAt = time.Now()

	if err := s.db.DB().Save(&dbAlgorithm).Error; err != nil {
		return nil, fmt.Errorf("failed to update algorithm: %w", err)
	}

	return modelToProto(&dbAlgorithm), nil
}

func (s *ManagementService) ListAlgorithms(ctx context.Context, req *v1.ListAlgorithmsRequest) (*v1.ListAlgorithmsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var dbAlgorithms []models.Algorithm
	if err := s.db.DB().Find(&dbAlgorithms).Error; err != nil {
		return nil, fmt.Errorf("failed to list algorithms: %w", err)
	}

	algorithms := make([]*v1.Algorithm, len(dbAlgorithms))
	for i, dbAlg := range dbAlgorithms {
		algorithms[i] = modelToProto(&dbAlg)
	}

	return &v1.ListAlgorithmsResponse{
		Algorithms: algorithms,
		Total:      int32(len(algorithms)),
	}, nil
}

func (s *ManagementService) GetAlgorithm(ctx context.Context, req *v1.GetAlgorithmRequest) (*v1.GetAlgorithmResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var dbAlgorithm models.Algorithm
	if err := s.db.DB().First(&dbAlgorithm, "id = ?", req.Id).Error; err != nil {
		return nil, fmt.Errorf("algorithm not found: %w", err)
	}

	var dbVersions []models.Version
	if err := s.db.DB().Where("algorithm_id = ?", req.Id).Order("version_number ASC").Find(&dbVersions).Error; err != nil {
		return nil, fmt.Errorf("failed to get versions: %w", err)
	}

	versions := make([]*v1.Version, len(dbVersions))
	for i, dbVer := range dbVersions {
		versions[i] = versionModelToProto(&dbVer)
	}

	return &v1.GetAlgorithmResponse{
		Algorithm: modelToProto(&dbAlgorithm),
		Versions:  versions,
	}, nil
}

func (s *ManagementService) CreateVersion(ctx context.Context, req *v1.CreateVersionRequest) (*v1.Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var dbAlgorithm models.Algorithm
	if err := s.db.DB().First(&dbAlgorithm, "id = ?", req.AlgorithmId).Error; err != nil {
		return nil, fmt.Errorf("algorithm not found: %w", err)
	}

	// 获取最新版本号
	var lastVersion models.Version
	nextVersionNumber := 1
	err := s.db.DB().Where("algorithm_id = ?", req.AlgorithmId).Order("version_number DESC").First(&lastVersion).Error
	if err == nil {
		nextVersionNumber = lastVersion.VersionNumber + 1
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

	dbVersion := &models.Version{
		ID:             fmt.Sprintf("ver_%d", time.Now().UnixNano()),
		AlgorithmID:    req.AlgorithmId,
		VersionNumber:  nextVersionNumber,
		MinioPath:      minioPath,
		SourceCodeFile: req.FileName,
		CommitMessage:  req.CommitMessage,
		CreatedAt:      time.Now(),
	}

	if err := s.db.DB().Create(dbVersion).Error; err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	// 更新算法的当前版本
	dbAlgorithm.CurrentVersionID = dbVersion.ID
	s.db.DB().Save(&dbAlgorithm)

	return versionModelToProto(dbVersion), nil
}

func (s *ManagementService) RollbackVersion(ctx context.Context, req *v1.RollbackVersionRequest) (*v1.Algorithm, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var dbAlgorithm models.Algorithm
	if err := s.db.DB().First(&dbAlgorithm, "id = ?", req.AlgorithmId).Error; err != nil {
		return nil, fmt.Errorf("algorithm not found: %w", err)
	}

	var dbVersion models.Version
	if err := s.db.DB().First(&dbVersion, "id = ? AND algorithm_id = ?", req.VersionId, req.AlgorithmId).Error; err != nil {
		return nil, fmt.Errorf("version not found: %w", err)
	}

	dbAlgorithm.CurrentVersionID = req.VersionId
	dbAlgorithm.UpdatedAt = time.Now()

	if err := s.db.DB().Save(&dbAlgorithm).Error; err != nil {
		return nil, fmt.Errorf("failed to rollback version: %w", err)
	}

	return modelToProto(&dbAlgorithm), nil
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

	// 数据库只保存路径，不保存完整URL
	dbPresetData := &models.PresetData{
		ID:        id,
		Filename:  req.Filename,
		Category:  req.Category,
		MinioPath: minioPath, // 只保存路径，如: preset-data/file.zip
		CreatedAt: time.Now(),
	}

	if err := s.db.DB().Create(dbPresetData).Error; err != nil {
		return nil, fmt.Errorf("failed to create preset data: %w", err)
	}

	// 返回时拼接完整URL
	scheme := "http"
	if s.cfg.MinIO.UseSSL {
		scheme = "https"
	}
	minioURL := fmt.Sprintf("%s://%s/%s/%s", scheme, s.cfg.MinIO.ExternalEndpoint, s.bucketName, minioPath)

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

	query := s.db.DB()
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}

	var dbPresetData []models.PresetData
	if err := query.Order("created_at DESC").Find(&dbPresetData).Error; err != nil {
		return nil, fmt.Errorf("failed to list preset data: %w", err)
	}

	files := make([]*v1.PresetData, len(dbPresetData))
	for i, dbData := range dbPresetData {
		files[i] = presetDataModelToProto(&dbData, scheme, s.cfg.MinIO.ExternalEndpoint, s.bucketName)
	}

	return &v1.ListPresetDataResponse{
		Files: files,
		Total: int32(len(files)),
	}, nil
}

func (s *ManagementService) DeletePresetData(ctx context.Context, req *v1.DeletePresetDataRequest) (*v1.DeletePresetDataResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var dbPresetData models.PresetData
	if err := s.db.DB().First(&dbPresetData, "id = ?", req.Id).Error; err != nil {
		return nil, fmt.Errorf("data not found: %w", err)
	}

	// 从MinIO删除文件
	if s.minioClient != nil {
		err := s.minioClient.RemoveObject(ctx, s.bucketName, dbPresetData.MinioPath, minio.RemoveObjectOptions{})
		if err != nil {
			fmt.Printf("Failed to remove object from MinIO: %v\n", err)
		}
	}

	// 从数据库删除
	if err := s.db.DB().Delete(&dbPresetData).Error; err != nil {
		return nil, fmt.Errorf("failed to delete preset data: %w", err)
	}

	return &v1.DeletePresetDataResponse{
		Success: true,
		Message: "Data deleted successfully",
	}, nil
}

func (s *ManagementService) GetPresetDataDownloadURL(ctx context.Context, fileID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var dbPresetData models.PresetData
	if err := s.db.DB().First(&dbPresetData, "id = ?", fileID).Error; err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	if s.minioClient == nil {
		return "", fmt.Errorf("minio client not available")
	}

	presignedURL, err := s.minioClient.PresignedGetObject(ctx, s.bucketName, dbPresetData.MinioPath, time.Hour*24, nil)
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

	// 数据库只保存路径，不保存完整URL
	dbPresetData := &models.PresetData{
		ID:        id,
		Filename:  filename,
		Category:  category,
		MinioPath: minioPath, // 只保存路径，如: preset-data/file.zip
		CreatedAt: time.Now(),
	}

	if err := s.db.DB().Create(dbPresetData).Error; err != nil {
		return nil, fmt.Errorf("failed to create preset data: %w", err)
	}

	// 返回时拼接完整URL
	scheme := "http"
	if s.cfg.MinIO.UseSSL {
		scheme = "https"
	}
	minioURL := fmt.Sprintf("%s://%s/%s/%s", scheme, s.cfg.MinIO.ExternalEndpoint, s.bucketName, minioPath)

	return &v1.UploadDataResponse{
		FileId:   id,
		MinioUrl: minioURL,
	}, nil
}

func (s *ManagementService) ListJobs(ctx context.Context, req *v1.ListJobsRequest) (*v1.ListJobsResponse, error) {
	var dbJobs []models.Job
	query := s.db.DB()

	if req.AlgorithmId != "" {
		query = query.Where("algorithm_id = ?", req.AlgorithmId)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	if err := query.Order("created_at DESC").Limit(100).Find(&dbJobs).Error; err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	jobs := make([]*v1.JobSummary, len(dbJobs))
	for i, dbJob := range dbJobs {
		jobs[i] = &v1.JobSummary{
			JobId:       dbJob.ID,
			AlgorithmId: dbJob.AlgorithmID,
			Status:      dbJob.Status,
			CreatedAt:   timestamppb.New(dbJob.CreatedAt),
		}
	}

	return &v1.ListJobsResponse{
		Jobs:  jobs,
		Total: int32(len(jobs)),
	}, nil
}

func (s *ManagementService) GetJobDetail(ctx context.Context, req *v1.GetJobDetailRequest) (*v1.JobDetail, error) {
	var dbJob models.Job
	if err := s.db.DB().First(&dbJob, "id = ?", req.JobId).Error; err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	return &v1.JobDetail{
		JobId:       dbJob.ID,
		AlgorithmId: dbJob.AlgorithmID,
		Mode:        dbJob.Mode,
		Status:      dbJob.Status,
		OutputUrl:   dbJob.OutputURL,
		LogUrl:      dbJob.LogURL,
		CreatedAt:   timestamppb.New(dbJob.CreatedAt),
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
