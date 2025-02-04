package env

import (
	"fmt"
	"os"
	"runtime"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
)

type Environment struct {
	NodeName  string
	NodeIP    string
	Namespace string
}

var Current Environment

func Load() error {
	nn, ok := os.LookupEnv(k8sconsts.NodeNameEnvVar)
	if !ok {
		return fmt.Errorf("env var %s is not set", k8sconsts.NodeNameEnvVar)
	}

	ni, ok := os.LookupEnv(k8sconsts.NodeIPEnvVar)
	if !ok {
		return fmt.Errorf("env var %s is not set", k8sconsts.NodeIPEnvVar)
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
