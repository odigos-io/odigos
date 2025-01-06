package env

import (
	"fmt"
	"os"
	"runtime"

	"github.com/odigos-io/odigos/common/consts"
)

const (
	NodeNameEnvVar = "NODE_NAME"
	NodeIPEnvVar   = "NODE_IP"
)

type Environment struct {
	NodeName  string
	NodeIP    string
	Namespace string
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

	ns, ok := os.LookupEnv(consts.CurrentNamespaceEnvVar)
	if !ok {
		return fmt.Errorf("env var %s is not set", consts.CurrentNamespaceEnvVar)
	}

	Current = Environment{
		NodeName:  nn,
		NodeIP:    ni,
		Namespace: ns,
	}
	return nil
}

func (e *Environment) IsEBPFSupported() bool {
	return runtime.GOOS == "linux"
}
