package service

import (
	"context"

	v1 "algorithm-platform/api/v1/proto"
)

type AlgorithmService struct {
	v1.UnimplementedAlgorithmServiceServer
}

func NewAlgorithmService() *AlgorithmService {
	return &AlgorithmService{}
}

func (s *AlgorithmService) ExecuteAlgorithm(ctx context.Context, req *v1.ExecuteRequest) (*v1.ExecuteResponse, error) {
	return &v1.ExecuteResponse{
		JobId:   "job_001",
		Status:  "pending",
		Message: "Algorithm execution queued",
	}, nil
}

func (s *AlgorithmService) GetJobStatus(ctx context.Context, req *v1.GetJobStatusRequest) (*v1.GetJobStatusResponse, error) {
	return &v1.GetJobStatusResponse{
		JobId:  req.JobId,
		Status: "running",
	}, nil
}
