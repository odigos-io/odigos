package agentenabled

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/podswebhook"
	"github.com/odigos-io/odigos/k8sutils/pkg/service"
)

// Mesh sidecar container names that install their own iptables redirect. Co-injecting
// odigos-browser-proxy would race on the same NAT rules and break traffic.
var meshSidecarContainerNames = map[string]struct{}{
	"istio-proxy":   {},
	"linkerd-proxy": {},
}

// injectBrowserProxy builds the odigos-browser-proxy sidecar and the iptables init container for a
// browser-instrumented web server container. Browser instrumentation does not run inside the pod,
// so the application container is left untouched: no env vars and no agent mounts are added to it.
// Instead the sidecar is placed in front of it (via the init container's iptables redirect) to
// inject the OpenTelemetry browser SDK <script> into served HTML and to proxy the browser's OTLP
// telemetry to the node-local collector.
//
// Returns the sidecar container, the init container, the agent directories that need to be copied
// (for the init-container mount method), and whether a pod volume mount is required. If the
// application container exposes no TCP port, injection is skipped (nil sidecar) since there is
// nothing to redirect.
func (p *PodsWebhook) injectBrowserProxy(
	logger logr.Logger,
	pod *corev1.Pod,
	appContainer *corev1.Container,
	namespace string,
	serviceName string,
	config common.OdigosConfiguration,
	distroMetadata *distro.OtelDistro,
) (*corev1.Container, *corev1.Container, map[string]struct{}, bool, error) {
	if mesh := meshSidecarInPod(pod); mesh != "" {
		logger.Info("browser instrumentation: skipping browser-proxy injection because a service-mesh sidecar is present (iptables redirect would collide)",
			"container", appContainer.Name, "meshSidecar", mesh)
		return nil, nil, nil, false, nil
	}

	appPort := firstTCPContainerPort(appContainer)
	if appPort == 0 {
		logger.Info("browser instrumentation: container has no TCP containerPort, skipping browser-proxy injection",
			"container", appContainer.Name)
		return nil, nil, nil, false, nil
	}

	mountMethod := common.K8sVirtualDeviceMountMethod
	if config.MountMethod != nil {
		mountMethod = *config.MountMethod
	}

	proxyImage := getBrowserProxyImage(config)
	agentDirResolved := strings.ReplaceAll(distroMetadata.BrowserSidecar.AgentDirectory, distro.AgentPlaceholderDirectory, k8sconsts.OdigosAgentsDirectory)

	falsePtr := false
	truePtr := true
	runAsProxy := k8sconsts.BrowserProxyRunAsUser
	runAsRoot := int64(0)

	// Same helper used by in-pod agents: on k8s >= 1.26 this resolves to the
	// odigos-data-collection-local-traffic ClusterIP service (InternalTrafficPolicy=Local).
	// On older clusters it falls back to http://$(NODE_IP):4318 — NODE_IP is injected below
	// only so that fallback can expand. User pods do not need hostNetwork.
	otlpEndpoint := service.LocalTrafficOTLPHttpDataCollectionEndpoint("$(NODE_IP)")

	probe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: k8sconsts.BrowserProxyHealthPath,
				Port: intstr.FromInt32(int32(k8sconsts.BrowserProxyListenPort)),
			},
		},
		InitialDelaySeconds: 1,
		PeriodSeconds:       10,
		TimeoutSeconds:      2,
		FailureThreshold:    3,
	}

	sidecar := corev1.Container{
		Name:            k8sconsts.BrowserProxyContainerName,
		Image:           proxyImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports: []corev1.ContainerPort{
			{ContainerPort: int32(k8sconsts.BrowserProxyListenPort)},
		},
		Env: []corev1.EnvVar{
			// NODE_IP must come first so the OTLP endpoint env var can reference $(NODE_IP)
			// when the LocalTraffic helper falls back to the node-IP form (k8s < 1.26).
			{
				Name:      "NODE_IP",
				ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.hostIP"}},
			},
			{Name: k8sconsts.BrowserProxyUpstreamEnvVar, Value: fmt.Sprintf("http://127.0.0.1:%d", appPort)},
			{Name: k8sconsts.BrowserProxyListenAddrEnvVar, Value: fmt.Sprintf(":%d", k8sconsts.BrowserProxyListenPort)},
			{Name: k8sconsts.BrowserProxyAgentDirEnvVar, Value: agentDirResolved},
			{Name: k8sconsts.BrowserProxyAgentFileEnvVar, Value: distroMetadata.BrowserSidecar.AgentFileName},
			{Name: k8sconsts.BrowserProxyOtlpHttpEndpointEnvVar, Value: otlpEndpoint},
			{Name: k8sconsts.BrowserProxyServiceNameEnvVar, Value: serviceName},
			{Name: k8sconsts.BrowserProxyResourceAttributesEnvVar, Value: fmt.Sprintf("k8s.namespace.name=%s", namespace)},
		},
		LivenessProbe:  probe,
		ReadinessProbe: probe.DeepCopy(),
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:                &runAsProxy,
			AllowPrivilegeEscalation: &falsePtr,
			Privileged:               &falsePtr,
		},
	}

	// Make the browser SDK bundle available to the sidecar, using the configured mount method.
	dirsToCopy := make(map[string]struct{})
	volumeMounted := false
	switch mountMethod {
	case common.K8sHostPathMountMethod, common.K8sInitContainerMountMethod, common.K8sCsiDriverMountMethod:
		podswebhook.MountDirectory(&sidecar, distroMetadata.BrowserSidecar.AgentDirectory)
		dirsToCopy[distroMetadata.BrowserSidecar.AgentDirectory] = struct{}{}
		volumeMounted = true
	case common.K8sVirtualDeviceMountMethod:
		podswebhook.InjectDeviceToContainer(&sidecar, k8sconsts.OdigosGenericDeviceName)
	}

	// The init container installs the iptables redirect (inbound app port -> sidecar) before the
	// application starts. It needs root + CAP_NET_ADMIN; the sidecar's own traffic (UID
	// BrowserProxyRunAsUser) is excluded from the redirect so it can reach the app on loopback.
	initContainer := corev1.Container{
		Name:            k8sconsts.BrowserProxyInitContainerName,
		Image:           proxyImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            []string{"init"},
		Env: []corev1.EnvVar{
			{Name: k8sconsts.BrowserProxyAppPortEnvVar, Value: strconv.Itoa(int(appPort))},
			{Name: k8sconsts.BrowserProxyUidEnvVar, Value: strconv.FormatInt(k8sconsts.BrowserProxyRunAsUser, 10)},
			{Name: k8sconsts.BrowserProxyListenAddrEnvVar, Value: fmt.Sprintf(":%d", k8sconsts.BrowserProxyListenPort)},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:                &runAsRoot,
			AllowPrivilegeEscalation: &truePtr,
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
		},
	}

	return &sidecar, &initContainer, dirsToCopy, volumeMounted, nil
}

func firstTCPContainerPort(container *corev1.Container) int32 {
	for _, port := range container.Ports {
		if port.ContainerPort > 0 && (port.Protocol == "" || port.Protocol == corev1.ProtocolTCP) {
			return port.ContainerPort
		}
	}
	return 0
}

func meshSidecarInPod(pod *corev1.Pod) string {
	for _, c := range pod.Spec.Containers {
		if _, ok := meshSidecarContainerNames[c.Name]; ok {
			return c.Name
		}
	}
	for _, c := range pod.Spec.InitContainers {
		if _, ok := meshSidecarContainerNames[c.Name]; ok {
			return c.Name
		}
	}
	return ""
}

func getBrowserProxyImage(config common.OdigosConfiguration) string {
	// In the installation/upgrade we set the image as env var, so prefer it when present.
	if img, ok := os.LookupEnv(k8sconsts.OdigosBrowserProxyEnvVarName); ok {
		return img
	}
	imageVersion := os.Getenv(consts.OdigosVersionEnvVarName)
	return config.ImagePrefix + "/" + k8sconsts.OdigosBrowserProxyImage + ":" + imageVersion
}
