// Package server is the importable frontend bootstrap. It exposes everything
// the OSS main binary needs to stand up the Odigos UI (`Flags`, `ParseFlags`,
// `Deps`, `Bootstrap`, `BuildRouter`, `ServeAndWait`), and is designed so an
// out-of-tree binary (e.g. odigos-enterprise's UI image) can import it and
// mount additional handlers via `RouterOpts.ExtraMounts` without duplicating
// the setup graph.
package server

import (
	"flag"

	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

// DefaultPort is the gin listener port matching the OSS service definition.
const DefaultPort = 3000

// Flags is the set of command-line options the frontend binary accepts.
// Kept here (vs. inlined in main) so out-of-tree wrappers reuse the exact
// same flag definitions and defaults.
type Flags struct {
	Version     bool
	Address     string
	Port        int
	Debug       bool
	KubeConfig  string
	KubeContext string
	Namespace   string
}

// ParseFlags reads the standard frontend flags from the default FlagSet and
// returns them. Out-of-tree mains may either call this verbatim or construct
// a `Flags` value directly.
func ParseFlags() Flags {
	defaultKubeConfig := env.GetDefaultKubeConfigPath()

	var flags Flags
	flag.BoolVar(&flags.Version, "version", false, "Print Odigos UI version.")
	flag.StringVar(&flags.Address, "address", "localhost", "Address to listen on")
	flag.IntVar(&flags.Port, "port", DefaultPort, "Port to listen on")
	flag.BoolVar(&flags.Debug, "debug", false, "Enable debug mode")
	flag.StringVar(&flags.KubeConfig, "kubeconfig", defaultKubeConfig, "Path to kubeconfig file")
	flag.StringVar(&flags.KubeContext, "kube-context", "", "Name of the kubeconfig context to use")
	flag.StringVar(&flags.Namespace, "namespace", env.GetCurrentNamespace(), "Kubernetes namespace where Odigos is installed")
	flag.Parse()
	return flags
}
