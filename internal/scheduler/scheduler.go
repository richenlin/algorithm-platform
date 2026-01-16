package scheduler

import (
	"context"
	"fmt"

	"algorithm-platform/pkg/docker"
)

type Scheduler struct {
	dockerClient *docker.Client
}

func New(dockerClient *docker.Client) *Scheduler {
	return &Scheduler{
		dockerClient: dockerClient,
	}
}

type JobConfig struct {
	Image       string
	AlgorithmID string
	JobID       string
	Env         map[string]string
	Mounts      []docker.Mount
	ResourceConfig
	TimeoutSeconds int
}

type ResourceConfig struct {
	CPULimit float64
	MemoryMB int
}

func (s *Scheduler) RunJob(ctx context.Context, cfg JobConfig) error {
	containerName := fmt.Sprintf("alg_%s_%s", cfg.AlgorithmID, cfg.JobID)

	env := make([]string, 0, len(cfg.Env))
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	dockerCfg := docker.ContainerConfig{
		Image:    cfg.Image,
		Env:      env,
		Mounts:   cfg.Mounts,
		CPULimit: cfg.CPULimit,
		MemoryMB: cfg.MemoryMB,
		Labels: map[string]string{
			"job_id":       cfg.JobID,
			"algorithm_id": cfg.AlgorithmID,
		},
	}

	containerID, err := s.dockerClient.CreateContainer(ctx, containerName, dockerCfg)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	if err := s.dockerClient.StartContainer(ctx, containerID); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

func (s *Scheduler) StopJob(ctx context.Context, jobID string) error {
	containers, err := s.dockerClient.ListContainers(ctx, map[string][]string{
		"label": {fmt.Sprintf("job_id=%s", jobID)},
	})
	if err != nil {
		return err
	}

	for _, c := range containers {
		if err := s.dockerClient.StopContainer(ctx, c.ID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Scheduler) RemoveJob(ctx context.Context, jobID string) error {
	containers, err := s.dockerClient.ListContainers(ctx, map[string][]string{
		"label": {fmt.Sprintf("job_id=%s", jobID)},
	})
	if err != nil {
		return err
	}

	for _, c := range containers {
		if err := s.dockerClient.RemoveContainer(ctx, c.ID, true); err != nil {
			return err
		}
	}

	return nil
}
