package env

import (
	"fmt"
	"os"
)

const (
	NodeNameEnvVar = "NODE_NAME"
)

type Environment struct {
	NodeName string
}

var Current Environment

func Load() error {
	nn, ok := os.LookupEnv(NodeNameEnvVar)
	if !ok {
		return fmt.Errorf("env var %s is not set", NodeNameEnvVar)
	}

	Current = Environment{
		NodeName: nn,
	}
	return nil
}
