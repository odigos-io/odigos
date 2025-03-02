package feature

import (
	"sync"

	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

type maturityLevel string

const (
	Alpha maturityLevel = "Alpha"
	Beta  maturityLevel = "Beta"
	GA    maturityLevel = "Stable"
)

var k8sVersion *version.Version

type featureSupport struct {
	alphaVersion string
	betaVersion  string
	gaVersion    string

	alphaSupported bool
	betaSupported  bool
	gaSupported    bool

	compareOnce sync.Once
}

func (fs *featureSupport) isEnabled(ml maturityLevel) bool {
	if k8sVersion == nil {
		return false
	}

	fs.compareOnce.Do(func() {
		if fs.alphaVersion == "" {
			fs.alphaSupported = false
		} else {
			fs.alphaSupported = k8sVersion.AtLeast(version.MustParse(fs.alphaVersion))
		}

		if fs.betaVersion == "" {
			fs.betaSupported = false
		} else {
			fs.betaSupported = k8sVersion.AtLeast(version.MustParse(fs.betaVersion))
		}

		if fs.gaVersion == "" {
			fs.gaSupported = false
		} else {
			fs.gaSupported = k8sVersion.AtLeast(version.MustParse(fs.gaVersion))
		}
	})

	switch ml {
	case Alpha:
		return fs.alphaSupported
	case Beta:
		return fs.betaSupported
	case GA:
		return fs.gaSupported
	default:
		return false
	}
}

// https://github.com/kubernetes/kubernetes/blob/v1.25.3/pkg/features/kube_features.go#L224
var (
	daemonSetUpdateSurge = &featureSupport{
		alphaVersion: "1.21.0",
		betaVersion:  "1.22.0",
		gaVersion:    "1.25.0",
	}

	DaemonSetUpdateSurge = func(ml maturityLevel) bool {
		return daemonSetUpdateSurge.isEnabled(ml)
	}
)

// https://github.com/kubernetes/kubernetes/blob/v1.26.0/pkg/features/kube_features.go#L775
var (
	serviceInternalTrafficPolicy = &featureSupport{
		alphaVersion: "1.21.0",
		betaVersion:  "1.22.0",
		gaVersion:    "1.26.0",
	}

	ServiceInternalTrafficPolicy = func(ml maturityLevel) bool {
		return serviceInternalTrafficPolicy.isEnabled(ml)
	}
)

// K8sVersion returns the Kubernetes version detected once Setup is called.
// It returns nil if Setup has not been called or failed.
//
// TODO: we should remove this function once the hpa.go file is updated to use featureSupport instances.
func K8sVersion() *version.Version {
	return k8sVersion
}

// Setup initializes the feature support based on the Kubernetes version.
// It should be called once before using any feature support.
// It must be called only for clients running inside a Kubernetes cluster.
func Setup() error {
	if k8sVersion != nil {
		return nil
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}

	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return err
	}

	parsedVersion, err := version.Parse(serverVersion.String())
	if err != nil || parsedVersion == nil {
		return err
	}

	k8sVersion = parsedVersion

	return nil
}
