package scheduler

import (
	"context"
	"fmt"
	"time"

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

func (s *Scheduler) GetJobStatus(ctx context.Context, jobID string) (string, int64, error) {
	containers, err := s.dockerClient.ListContainers(ctx, map[string][]string{
		"label": {fmt.Sprintf("job_id=%s", jobID)},
	})
	if err != nil {
		return "", -1, err
	}

	if len(containers) == 0 {
		return "not_found", -1, nil
	}

	containerID := containers[0].ID
	status, err := s.dockerClient.GetContainerStatus(ctx, containerID)
	if err != nil {
		return "", -1, err
	}

	state := "unknown"
	if status.State != nil {
		if status.State.Running {
			state = "running"
		} else if status.State.Status == "exited" {
			state = "exited"
			return state, int64(status.State.ExitCode), nil
		}
	}

	return state, 0, nil
}

func (s *Scheduler) CleanUp(ctx context.Context, olderThan time.Duration) error {
	filters := map[string][]string{
		"label": {"algorithm_platform=1"},
	}

	containers, err := s.dockerClient.ListContainers(ctx, filters)
	if err != nil {
		return err
	}

	for _, c := range containers {
		if time.Since(time.Unix(c.Created, 0)) > olderThan {
			if c.Status == "exited" || c.State == "exited" {
				if err := s.dockerClient.RemoveContainer(ctx, c.ID, true); err != nil {
					continue
				}
			}
		}
	}

	return nil
}
