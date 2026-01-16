package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

type Client struct {
	cli *client.Client
}

func New(host string) (*Client, error) {
	cli, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &Client{cli: cli}, nil
}

type ContainerConfig struct {
	Image      string
	Cmd        []string
	Env        []string
	WorkingDir string
	Labels     map[string]string
	CPULimit   float64
	MemoryMB   int
	Timeout    int
	Mounts     []Mount
}

type Mount struct {
	Type     string
	Source   string
	Target   string
	ReadOnly bool
}

func (c *Client) CreateContainer(ctx context.Context, name string, cfg ContainerConfig) (string, error) {
	hostConfig := &container.HostConfig{
		Mounts: make([]mount.Mount, len(cfg.Mounts)),
	}

	if cfg.CPULimit > 0 {
		hostConfig.NanoCPUs = int64(cfg.CPULimit * 1e9)
	}

	if cfg.MemoryMB > 0 {
		hostConfig.Memory = int64(cfg.MemoryMB * 1024 * 1024)
	}

	for i, m := range cfg.Mounts {
		hostConfig.Mounts[i] = mount.Mount{
			Type:     mount.Type(m.Type),
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		}
	}

	resp, err := c.cli.ContainerCreate(ctx, &container.Config{
		Image:      cfg.Image,
		Cmd:        cfg.Cmd,
		Env:        cfg.Env,
		WorkingDir: cfg.WorkingDir,
		Labels:     cfg.Labels,
		Tty:        false,
	}, hostConfig, nil, nil, name)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (c *Client) StartContainer(ctx context.Context, id string) error {
	return c.cli.ContainerStart(ctx, id, container.StartOptions{})
}

func (c *Client) StopContainer(ctx context.Context, id string) error {
	timeout := int(10)
	return c.cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout})
}

func (c *Client) RemoveContainer(ctx context.Context, id string, force bool) error {
	return c.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: force})
}

func (c *Client) GetContainerLogs(ctx context.Context, id string) (io.ReadCloser, error) {
	return c.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false,
		Timestamps: false,
	})
}

func (c *Client) GetContainerStatus(ctx context.Context, id string) (container.InspectResponse, error) {
	return c.cli.ContainerInspect(ctx, id)
}

func (c *Client) ListContainers(ctx context.Context, filterLabels map[string][]string) ([]types.Container, error) {
	f := filters.NewArgs()
	for k, vals := range filterLabels {
		for _, v := range vals {
			f.Add(k, v)
		}
	}

	return c.cli.ContainerList(ctx, container.ListOptions{Filters: f})
}

func (c *Client) PullImage(ctx context.Context, imageRef string) error {
	reader, err := c.cli.ImagePull(ctx, imageRef, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(io.Discard, reader)
	return err
}

func (c *Client) WaitContainer(ctx context.Context, id string) (int64, error) {
	statusCh, errCh := c.cli.ContainerWait(ctx, id, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			return -1, err
		}
	case status := <-statusCh:
		return status.StatusCode, nil
	}

	return -1, fmt.Errorf("wait failed")
}
