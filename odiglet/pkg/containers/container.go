package containers

import (
	"errors"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	v1 "k8s.io/api/core/v1"
	"strings"
)

type ContainerID struct {
	Runtime string
	ID      string
}

func newContainerID(str string) (*ContainerID, error) {
	parts := strings.Split(str, "://")
	if len(parts) != 2 {
		return nil, errors.New("invalid container id")
	}

	return &ContainerID{
		Runtime: parts[0],
		ID:      parts[1],
	}, nil
}

func FindIDs(pod *v1.Pod) ([]*ContainerID, error) {
	// Get container ids of found containers
	var containerIds []*ContainerID
	for _, container := range pod.Status.ContainerStatuses {
		id, err := newContainerID(container.ContainerID)
		if err != nil {
			return nil, err
		}

		containerIds = append(containerIds, id)
	}

	log.Logger.V(0).Info("Found container ids", "containerIds", containerIds)
	return containerIds, nil
}
