package runtimes

import (
	"context"
	"errors"
	docker "github.com/docker/docker/client"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
)

type dockerRuntime struct{}

func (d *dockerRuntime) Name() ContainerRuntime {
	return Docker
}

func (d *dockerRuntime) GetFileSystemPath(containerId string) (string, error) {
	// Init docker client
	client, err := docker.NewClientWithOpts(docker.FromEnv)
	if err != nil {
		log.Logger.Error(err, "Failed to init docker client")
		return "", err
	}

	// Inspect container and print response
	defer client.Close()
	inspect, err := client.ContainerInspect(context.Background(), containerId)
	if err != nil {
		log.Logger.Error(err, "Failed to inspect container")
		return "", err
	}

	fs, exists := inspect.GraphDriver.Data["MergedDir"]
	if !exists {
		err = errors.New("MergeDir not found in GraphDriver.Data")
		log.Logger.Error(err, "Failed to get filesystem path")
		return "", err
	}

	return fs, nil
}
