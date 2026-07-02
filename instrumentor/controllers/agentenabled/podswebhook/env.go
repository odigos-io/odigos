package podswebhook

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros/distro"
	containersutil "github.com/odigos-io/odigos/k8sutils/pkg/containers"
	"github.com/odigos-io/odigos/k8sutils/pkg/service"
	corev1 "k8s.io/api/core/v1"
)

type EnvVarNamesMap map[string]struct{}

func injectEnvVarObjectFieldRefToPodContainer(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVarName, envVarRef string) EnvVarNamesMap {
	if _, exists := (existingEnvNames)[envVarName]; exists {
		return existingEnvNames
	}

	container.Env = append(container.Env, corev1.EnvVar{
		Name: envVarName,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: envVarRef,
			},
		},
	})
	existingEnvNames[envVarName] = struct{}{}
	return existingEnvNames
}

func InjectConstEnvVarToPodContainer(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVarName, envVarValue string) EnvVarNamesMap {
	if _, exists := existingEnvNames[envVarName]; exists {
		return existingEnvNames
	}
	container.Env = append(container.Env, corev1.EnvVar{
		Name:  envVarName,
		Value: envVarValue,
	})
	existingEnvNames[envVarName] = struct{}{}
	return existingEnvNames
}

// jdkJavaOptionsEnvVar carries the Java memory-profiling startup flags. We use
// JDK_JAVA_OPTIONS (honored by JDK 9+, applied additively alongside
// JAVA_TOOL_OPTIONS) so we never disturb the tracing agent's JAVA_TOOL_OPTIONS.
const jdkJavaOptionsEnvVar = "JDK_JAVA_OPTIONS"

// javaJFRMemoryFlags starts a continuous Flight Recording at JVM init with the
// leak profiler (jdk.OldObjectSample) initialized. old-object tracking can only
// be set up at startup — a runtime attach cannot enable it — which is why memory
// leak profiling requires this startup flag (a one-time pod restart). The agent
// then periodically dumps the "odigos" recording for alloc + leak signals.
const javaJFRMemoryFlags = "-XX:StartFlightRecording=name=odigos,settings=profile,maxsize=100m " +
	"-XX:FlightRecorderOptions=old-object-queue-size=256"

// InjectJavaMemoryProfiling enables the JVM-side startup recording the JFR memory
// engine reads. No-op if the container already sets JDK_JAVA_OPTIONS.
func InjectJavaMemoryProfiling(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	return InjectConstEnvVarToPodContainer(existingEnvNames, container, jdkJavaOptionsEnvVar, javaJFRMemoryFlags)
}

const (
	ldPreloadEnvVar  = "LD_PRELOAD"
	mallocConfEnvVar = "MALLOC_CONF"
	// useZendAllocEnvVar=0 makes PHP route ZendMM/emalloc allocations through the
	// system malloc; without it PHP userland allocations stay in Zend's arena and
	// are invisible to the libmemsample interposer (so only C-extension mallocs —
	// TLS/cert parsing — get captured, which looks like noise). pythonMallocEnvVar
	// =malloc does the same for CPython's pymalloc small-object arena. Both route
	// the interpreter's own allocations to malloc so the interposer can attribute
	// them to interpreter (file:function:line) frames. They cost some throughput
	// and are only set when memory profiling is enabled for that container.
	useZendAllocEnvVar = "USE_ZEND_ALLOC"
	pythonMallocEnvVar = "PYTHONMALLOC"
	// jemallocProfSoPath is the prof-enabled jemalloc the odiglet delivers; for
	// glibc/default C/C++/Rust we preload it so the allocator profiles itself
	// (Poisson-sampled, real live-heap, out-of-band dumps) — the production model,
	// not a home-grown malloc shim.
	// Canonical name odiglet also stages the prof jemalloc under; the out-of-process
	// native reader greps process maps for "/libjemalloc.so", so the LD_PRELOAD path
	// must carry that name (not "-prof") for the allocator to be detected.
	jemallocProfSoPath = "/var/odigos/memprof/libjemalloc.so"
	// jemallocProfConf enables jemalloc's heap profiler: Poisson sampling at
	// 2^19=512KiB (lg_prof_sample), cumulative accounting, auto-dump every
	// 2^24=16MiB allocated (lg_prof_interval) to a prefix the agent reads
	// out-of-process via /proc/<pid>/root.
	// lg_prof_sample:18 = 256 KiB Poisson sample (matches the default sampleSizeBytes);
	// lg_prof_interval:20 = auto-dump every 1 MiB allocated. History: 16 MiB meant
	// low-allocation services never produced a first dump (native looked empty in
	// quiet/demo clusters); 2 MiB improved that but a low-traffic C/C++ service
	// (e.g. a shipping-quote endpoint) could still go many minutes between dumps, so
	// it rendered intermittently. 1 MiB makes even quiet native services dump on a
	// steady cadence so they appear reliably, while still bounding overhead — each
	// dump is small and the agent reaps it after reading. The reader unbiases the
	// sampled counts regardless of interval.
	jemallocProfConf = "prof:true,prof_active:true,prof_accum:true,lg_prof_sample:18,lg_prof_interval:20,prof_prefix:/tmp/odigos-jeprof"
	// libmemsampleMuslSoPath is the musl-built sampling interposer the odiglet
	// delivers. We preload it into musl containers (Alpine/scratch) where the
	// glibc jemalloc-prof lib cannot be loaded — it instruments the default
	// (musl) allocator directly and writes the same heap_v2 dumps the agent reads.
	libmemsampleMuslSoPath = "/var/odigos/memprof/libmemsample-musl.so"
	// libmemsampleSoPath is the glibc-built sampling interposer. Preloaded into
	// interpreted runtimes (CPython/Ruby/PHP) so the interpreter's own glibc malloc
	// is sampled and dumped for the agent — the same heap_v2 dumps the native reader
	// consumes, but with interpreter-aware frames.
	libmemsampleSoPath = "/var/odigos/memprof/libmemsample.so"
)

// InjectInterpretedMemoryProfiling LD_PRELOADs the libmemsample malloc interposer
// into a Python/Ruby/PHP container so allocations made by the interpreter are
// sampled. libc-aware exactly like the native path: the glibc lib for glibc, the
// musl lib for musl, nothing for unknown libc (a wrong preload aborts musl). The
// dumps land at the libmemsample default prefix the agent's glibc-native reader
// already discovers, so no MALLOC_CONF is needed.
func InjectInterpretedMemoryProfiling(existingEnvNames EnvVarNamesMap, container *corev1.Container, libc *common.LibCType, lang common.ProgrammingLanguage) EnvVarNamesMap {
	preloaded := false
	switch {
	case libc != nil && *libc == common.Musl:
		// musl interpreted: SKIP the preload. The musl-built interposer loads cleanly
		// (no loader abort), but is not yet memory-safe under a real Alpine interpreter
		// that drives heavy C++-extension allocation (observed std::bad_alloc / SIGSEGV
		// on Alpine CPython + gRPC C-extensions with PYTHONMALLOC=malloc). Until the
		// musl interposer is hardened, we MUST NOT crash the app: a tagged-musl
		// interpreter runs clean without memory profiling. (glibc interpreted is the
		// common, validated path; musl interpreted is a rare edge.) Re-enable by
		// restoring the libmemsampleMuslSoPath preload once the musl lib is solid.
		_ = libmemsampleMuslSoPath
	default:
		// glibc OR unknown: use the glibc interposer. Unlike the native C/C++ path
		// (where unknown libc gets no preload to avoid aborting a musl process), the
		// interpreted runtimes we target — CPython, MRI Ruby, PHP — ship overwhelmingly
		// as glibc (Debian/Ubuntu) images, and runtime libc detection often can't tag
		// the interpreter binary. Defaulting unknown to glibc lets these profile out of
		// the box; an interpreter that is actually musl is tagged by libc detection on a
		// subsequent cycle and then takes the skip branch above (no preload, no crash).
		existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, ldPreloadEnvVar, libmemsampleSoPath)
		preloaded = true
	}
	// Route the interpreter's own allocations through malloc so the interposer can see
	// and attribute them. This is ONLY meaningful when the interposer is actually
	// preloaded: with no preload (the musl skip above) these env vars just change the
	// allocator for no profiling benefit and add overhead, so we gate them on preloaded.
	if preloaded {
		switch lang {
		case common.PhpProgrammingLanguage:
			existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, useZendAllocEnvVar, "0")
		case common.PythonProgrammingLanguage:
			existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, pythonMallocEnvVar, "malloc")
			// Ruby: heap objects come from GC pages with no safe "use malloc" switch, but
			// strings/arrays/buffers already go through ruby_xmalloc -> malloc, which the
			// interposer captures via rb_profile_frames. No extra env needed.
		}
	}
	return existingEnvNames
}

// NativeMemoryPreloads reports whether InjectNativeMemoryProfiling will LD_PRELOAD
// a lib (and therefore the caller must mount the /var/odigos/memprof dir). True for
// both known glibc and known musl; false for unknown libc (where preload is unsafe).
func NativeMemoryPreloads(libc *common.LibCType) bool {
	return libc != nil && (*libc == common.Glibc || *libc == common.Musl)
}

// InjectNativeMemoryProfiling enables allocator-integrated heap profiling for a
// C/C++/Rust container. The mechanism is chosen by libc:
//
//   - glibc: LD_PRELOAD prof-enabled jemalloc + MALLOC_CONF — the allocator samples
//     its own fast path (Poisson, real live-heap) and writes dumps the agent reads.
//   - musl:  LD_PRELOAD the musl-built libmemsample interposer, which samples the
//     default musl allocator and writes the same heap_v2 dumps. (jemalloc-prof is a
//     glibc binary and would abort a musl process, so we use the musl-safe lib.)
//   - unknown libc: NO preload — only MALLOC_CONF, which is a no-op unless the app
//     already links jemalloc-prof. Never risk a crash on an unidentified loader.
//
// crash-safety: the two loaders disagree on a failed preload — glibc's ld.so warns
// and continues, but musl's loader ABORTS. We therefore only ever preload a lib
// that matches the detected libc; for unknown libc we preload nothing. No-op for any
// var the container already sets (e.g. an app with its own allocator).
func InjectNativeMemoryProfiling(existingEnvNames EnvVarNamesMap, container *corev1.Container, libc *common.LibCType) EnvVarNamesMap {
	switch {
	case libc != nil && *libc == common.Glibc:
		existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, ldPreloadEnvVar, jemallocProfSoPath)
		existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, mallocConfEnvVar, jemallocProfConf)
	case libc != nil && *libc == common.Musl:
		// musl-safe interposer; MALLOC_CONF is jemalloc-specific so it is omitted here.
		existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, ldPreloadEnvVar, libmemsampleMuslSoPath)
	default:
		// Unknown libc: no preload. MALLOC_CONF alone is safe and a no-op unless the
		// app already links a prof-enabled jemalloc.
		existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, mallocConfEnvVar, jemallocProfConf)
	}
	return existingEnvNames
}

func InjectTemplatedEnvVarToPodContainer(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVarName string, envVarValueTemplate *template.Template, distroParams map[string]string) (EnvVarNamesMap, error) {
	if _, exists := existingEnvNames[envVarName]; exists {
		return existingEnvNames, nil
	}

	var buf bytes.Buffer
	err := envVarValueTemplate.Execute(&buf, distroParams)
	if err != nil {
		// Should not happen. values are statically used from distro manifest which should be tested.
		return existingEnvNames, err
	}
	templatedEnvVarValue := buf.String()

	container.Env = append(container.Env, corev1.EnvVar{
		Name:  envVarName,
		Value: templatedEnvVarValue,
	})

	existingEnvNames[envVarName] = struct{}{}
	return existingEnvNames, nil
}

func injectNodeIpEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	return injectEnvVarObjectFieldRefToPodContainer(existingEnvNames, container, k8sconsts.NodeIPEnvVar, "status.hostIP")
}

func InjectOdigosK8sEnvVars(existingEnvNames EnvVarNamesMap, container *corev1.Container, distroName string, ns string) EnvVarNamesMap {
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarContainerName, container.Name)
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarDistroName, distroName)
	existingEnvNames = injectEnvVarObjectFieldRefToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarPodName, "metadata.name")
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarNamespace, ns)
	return existingEnvNames
}

func InjectOpampServerEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	existingEnvNames = injectNodeIpEnvVar(existingEnvNames, container)
	opAmpServerHost := service.LocalTrafficOpAmpOdigletEndpoint("$(NODE_IP)")
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, commonconsts.OpampServerHostEnvName, opAmpServerHost)
	return existingEnvNames
}

// InjectOpampUnixSocketEnvVar sets ODIGOS_OPAMP_UNIX_SOCKET (Unix OpAMP; no ODIGOS_OPAMP_SERVER_HOST).
func InjectOpampUnixSocketEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	return InjectConstEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OpampUnixSocketEnvName, k8sconsts.OdigosOpampUnixSocketPath)
}

func InjectOtlpHttpEndpointEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	existingEnvNames = injectNodeIpEnvVar(existingEnvNames, container)
	otlpHttpEndpoint := service.LocalTrafficOTLPHttpDataCollectionEndpoint("$(NODE_IP)")
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelExporterEndpointEnvName, otlpHttpEndpoint)
	return existingEnvNames
}

func InjectStaticEnvVarsToPodContainer(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVars []distro.StaticEnvironmentVariable, distroParams map[string]string) (EnvVarNamesMap, error) {
	for _, envVar := range envVars {
		if envVar.Template == nil {
			existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, envVar.EnvName, envVar.EnvValue)
		} else {
			var err error // make sure we don't shadow the error or the existingEnvNames
			existingEnvNames, err = InjectTemplatedEnvVarToPodContainer(existingEnvNames, container, envVar.EnvName, envVar.Template, distroParams)
			if err != nil {
				return existingEnvNames, fmt.Errorf("failed to inject static environment variable %s: %w", envVar.EnvName, err)
			}
		}
	}
	return existingEnvNames, nil
}

func signalOtlpExporterEnvValue(enabled bool) string {
	if enabled {
		return "otlp"
	}
	return "none"
}

func InjectSignalsAsStaticOtelEnvVars(existingEnvNames EnvVarNamesMap, container *corev1.Container, tracesEnabled bool, metricsEnabled bool, logsEnabled bool) EnvVarNamesMap {

	logsExporter := signalOtlpExporterEnvValue(logsEnabled)
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelLogsExporter, logsExporter)

	metricsExporter := signalOtlpExporterEnvValue(metricsEnabled)
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelMetricsExporter, metricsExporter)

	tracesExporter := signalOtlpExporterEnvValue(tracesEnabled)
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelTracesExporter, tracesExporter)

	return existingEnvNames
}

func InjectUserEnvForLang(odigosConfiguration *common.OdigosConfiguration, pod *corev1.Pod, ic *odigosv1.InstrumentationConfig) {
	languageSpecificEnvs := odigosConfiguration.UserInstrumentationEnvs.Languages

	// Check for conatiner language and inject env vars if they not exists
	for _, containerDetailes := range ic.Status.RuntimeDetailsByContainer {
		langConfig, exists := languageSpecificEnvs[containerDetailes.Language]
		if !exists || !langConfig.Enabled {
			continue
		}

		container := containersutil.GetContainerByName(pod.Spec.Containers, containerDetailes.ContainerName)
		if container == nil {
			continue
		}
		existingEnvNames := GetEnvVarNamesSet(container)

		for envName, envValue := range langConfig.EnvVars {
			existingEnvNames = InjectConstEnvVarToPodContainer(
				existingEnvNames,
				container,
				envName,
				envValue,
			)
		}
	}
}

// Create a set of existing environment variable names
// to avoid duplicates when injecting new environment variables
// into the container.
func GetEnvVarNamesSet(container *corev1.Container) EnvVarNamesMap {
	envSet := make(EnvVarNamesMap, len(container.Env))
	for _, envVar := range container.Env {
		envSet[envVar.Name] = struct{}{}
	}
	return envSet
}

func InjectAgentDiagnosticsEnvVars(existingEnvNames EnvVarNamesMap, container *corev1.Container, agentDiagnostics *instrumentationrules.AgentDiagnostics) EnvVarNamesMap {
	if agentDiagnostics == nil {
		return existingEnvNames
	}
	if agentDiagnostics.OdigosLogLevel != nil {
		existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, commonconsts.OdigosLogLevelEnvVarName, agentDiagnostics.OdigosLogLevel.EnvVarValue())
	}
	if agentDiagnostics.OpenTelemetryComponentsLogLevel != nil {
		existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelLogLevelEnvVarName, agentDiagnostics.OpenTelemetryComponentsLogLevel.EnvVarValue())
	}
	return existingEnvNames
}
