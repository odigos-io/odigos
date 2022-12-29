package runtimes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	cri "k8s.io/cri-api/pkg/apis/runtime/v1"
)

const (
	runtimeEndpoint = "unix:///run/crio/crio.sock"
	infoKey         = "info"
)

var (
	ErrMountAnnotationNotFound = errors.New("mount path annotation not found")
)

type crioRuntime struct{}

func (c *crioRuntime) Name() ContainerRuntime {
	return Crio
}

func (c *crioRuntime) GetFileSystemPath(containerId string) (string, error) {
	// Create connection cri endpoint
	conn, err := grpc.Dial(runtimeEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Logger.Error(err, "Failed to connect to cri endpoint")
		return "", err
	}

	defer conn.Close()
	client := cri.NewRuntimeServiceClient(conn)
	res, err := client.ContainerStatus(context.Background(), &cri.ContainerStatusRequest{ContainerId: containerId, Verbose: true})
	if err != nil {
		log.Logger.Error(err, "Failed to get container status")
		return "", err
	}

	infoMap := res.GetInfo()
	infoJsonRaw, exists := infoMap[infoKey]
	if !exists {
		err = ErrMountAnnotationNotFound
		log.Logger.Error(err, "No key 'info' in container info map")
		return "", err
	}

	// Parse infoJsonRaw to json
	var infoJson map[string]interface{}
	err = json.Unmarshal([]byte(infoJsonRaw), &infoJson)
	if err != nil {
		log.Logger.Error(err, "Failed to parse container info json")
		return "", err
	}

	// Get pid from infoJson and use it to get the mount path
	pidRaw, exists := infoJson["pid"]
	if !exists {
		err = ErrMountAnnotationNotFound
		log.Logger.Error(err, "No key 'pid' in container info json")
		return "", err
	}

	pid := int(pidRaw.(float64))
	return fmt.Sprintf("/proc/%d/root", pid), nil
}
