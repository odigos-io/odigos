package env

import (
	"fmt"
	"os"
	"runtime"
)

const (
	NodeNameEnvVar = "NODE_NAME"
	NodeIPEnvVar   = "NODE_IP"
)

type Environment struct {
	NodeName string
	NodeIP   string
}

var Current Environment

func Load() error {
	nn, ok := os.LookupEnv(NodeNameEnvVar)
	if !ok {
		return fmt.Errorf("env var %s is not set", NodeNameEnvVar)
	}

	ni, ok := os.LookupEnv(NodeIPEnvVar)
	if !ok {
		return fmt.Errorf("env var %s is not set", NodeIPEnvVar)
	}

	Current = Environment{
		NodeName: nn,
		NodeIP:   ni,
	}
	return nil
}

func (e *Environment) IsEBPFSupported() bool {
	return runtime.GOOS == "linux"
}
