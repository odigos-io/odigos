package runtimes

import "errors"

type ContainerRuntime string

const (
	Docker     ContainerRuntime = "docker"
	Containerd ContainerRuntime = "containerd"
	Crio       ContainerRuntime = "cri-o"
)

type Runtime interface {
	Name() ContainerRuntime
	GetFileSystemPath(containerId string) (string, error)
}

var (
	availableRuntimes  = []Runtime{&containerdRuntime{}}
	ErrRuntimeNotFound = errors.New("runtime not found")
	runtimesMap        = calculateRuntimesMap()
)

func calculateRuntimesMap() map[ContainerRuntime]Runtime {
	result := make(map[ContainerRuntime]Runtime)
	for _, runtime := range availableRuntimes {
		result[runtime.Name()] = runtime
	}

	return result
}

func ByName(name string) (Runtime, error) {
	runtime, ok := runtimesMap[ContainerRuntime(name)]
	if !ok {
		return nil, ErrRuntimeNotFound
	}

	return runtime, nil
}
