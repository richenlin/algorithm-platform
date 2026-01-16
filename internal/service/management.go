package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "algorithm-platform/api/v1/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ManagementService struct {
	v1.UnimplementedManagementServiceServer

	mu         sync.RWMutex
	algorithms map[string]*v1.Algorithm
	versions   map[string][]*v1.Version
}

func NewManagementService() *ManagementService {
	return &ManagementService{
		algorithms: make(map[string]*v1.Algorithm),
		versions:   make(map[string][]*v1.Version),
	}
}

func (s *ManagementService) CreateAlgorithm(ctx context.Context, req *v1.CreateAlgorithmRequest) (*v1.Algorithm, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("alg_%d", time.Now().UnixNano())

	algorithm := &v1.Algorithm{
		Id:          id,
		Name:        req.Name,
		Description: req.Description,
		Language:    req.Language,
		Platform:    req.Platform,
		Category:    req.Category,
		Entrypoint:  req.Entrypoint,
		CreatedAt:   timestamppb.Now(),
		UpdatedAt:   timestamppb.Now(),
	}

	s.algorithms[id] = algorithm

	return algorithm, nil
}

func (s *ManagementService) UpdateAlgorithm(ctx context.Context, req *v1.UpdateAlgorithmRequest) (*v1.Algorithm, error) {
	return &v1.Algorithm{
		Id:   req.Id,
		Name: req.Name,
	}, nil
}

func (s *ManagementService) ListAlgorithms(ctx context.Context, req *v1.ListAlgorithmsRequest) (*v1.ListAlgorithmsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	algorithms := make([]*v1.Algorithm, 0, len(s.algorithms))
	for _, alg := range s.algorithms {
		algorithms = append(algorithms, alg)
	}

	return &v1.ListAlgorithmsResponse{
		Algorithms: algorithms,
		Total:      int32(len(algorithms)),
	}, nil
}

func (s *ManagementService) CreateVersion(ctx context.Context, req *v1.CreateVersionRequest) (*v1.Version, error) {
	return &v1.Version{
		Id:            "v1",
		AlgorithmId:   req.AlgorithmId,
		VersionNumber: 1,
	}, nil
}

func (s *ManagementService) RollbackVersion(ctx context.Context, req *v1.RollbackVersionRequest) (*v1.Algorithm, error) {
	return &v1.Algorithm{
		Id:               req.AlgorithmId,
		CurrentVersionId: req.VersionId,
	}, nil
}

func (s *ManagementService) UploadPresetData(ctx context.Context, req *v1.UploadDataRequest) (*v1.UploadDataResponse, error) {
	return &v1.UploadDataResponse{
		FileId:   "data_001",
		MinioUrl: "http://minio.local/bucket/" + req.Filename,
	}, nil
}

func (s *ManagementService) ListPresetData(ctx context.Context, req *v1.ListPresetDataRequest) (*v1.ListPresetDataResponse, error) {
	return &v1.ListPresetDataResponse{
		Files: []*v1.PresetData{},
		Total: 0,
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
