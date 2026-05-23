package process

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/odigos-io/odigos/common/logger"

	"github.com/opencontainers/cgroups"
	"github.com/opencontainers/cgroups/systemd"
)

const (
	defaultCgroupRoot = "/sys/fs/cgroup"
	cgroupRootEnvVar  = "ODIGOS_CGROUP_ROOT"
	selfCgroupPath    = "/proc/self/cgroup"
)

// sysCgroupRoot returns the root path where the host cgroupfs is visible
// inside this process. Reads ODIGOS_CGROUP_ROOT, falling back to
// /sys/fs/cgroup for non-Kubernetes deployments where the binary runs
// directly on a node.
func sysCgroupRoot() string {
	if v := os.Getenv(cgroupRootEnvVar); v != "" {
		return v
	}
	return defaultCgroupRoot
}

// cgroupLayout is the host-level cgroup configuration. It is probed
// once at package init and cached.
type cgroupLayout struct {
	Systemd            bool
	Valid              bool
	KubepodsPrefix     string
	SystemdScopePrefix string
	root               string
}

var (
	// hostCgroupLayout is detected once at DiscoverCgroupLayout.
	hostCgroupLayout cgroupLayout
	isCgroupV2       = cgroups.IsCgroup2UnifiedMode()
	initOnce         sync.Once
)

func DiscoverCgroupLayout() {
	initOnce.Do(func() {
		log := logger.LoggerCompat().With("subsystem", "cgroup")

		// currently we only support the fast cgroup-based PID resolution on cgroup v2.
		// Supporting v1 adds additional complexity for mounting the cgroup fs.
		// cgroup v1 uses many per-controller mounts that need HostToContainer propagation,
		// which only works if the host marked /sys/fs/cgroup as shared — not always true, and forcing it requires a privileged init container that mutates host state.
		if !isCgroupV2 {
			log.Info("detected cgroup v1, the fast path for PID resolution will be disabled")
			return
		}

		hostCgroupLayout.root = sysCgroupRoot()

		// based on our self cgroup paths,
		// try to detect if we're running in a systemd-based cgroup hierarchy, and locate the kubepods root.
		if parsed, err := cgroups.ParseCgroupFile(selfCgroupPath); err == nil {
			for _, p := range parsed {
				// example for systemd path:
				// /kubelet.slice/kubelet-kubepods.slice/kubelet-kubepods-burstable.slice/kubelet-kubepods-burstable-pod<UID>.slice/cri-containerd-<UID>.scope
				if strings.Contains(p, ".slice/") || strings.HasSuffix(p, ".scope") {
					hostCgroupLayout.Systemd = true
					if sp := extractSystemdScopePrefix(p); sp != "" {
						hostCgroupLayout.SystemdScopePrefix = sp
					}
				}
				if prefix, ok := extractKubepodsPrefix(p); ok {
					hostCgroupLayout.KubepodsPrefix = prefix
					hostCgroupLayout.Valid = true
					break
				}
			}
		}

		// selfCgroupPath gives the cgroup-ns-relativized view. When
		// we run in a private cgroup namespace (privileged: false) we can't locate the kubepods root from it.
		// Fall back to probing well-known paths under the mounted host cgroupfs.
		if !hostCgroupLayout.Valid {
			probeLayoutFromFS(&hostCgroupLayout)
		}

		log.Info("host cgroup layout detected",
			"valid", hostCgroupLayout.Valid,
			"systemd", hostCgroupLayout.Systemd,
			"kubepodsPrefix", hostCgroupLayout.KubepodsPrefix,
			"systemdScopePrefix", hostCgroupLayout.SystemdScopePrefix,
			"cgroupRoot", hostCgroupLayout.root,
		)
	})
}

func probeLayoutFromFS(l *cgroupLayout) {
	root := l.root
	candidates := []struct {
		path    string
		systemd bool
		prefix  string
	}{
		{filepath.Join(root, "kubepods.slice"), true, ""},
		{filepath.Join(root, "kubelet.slice", "kubelet-kubepods.slice"), true, "kubelet"},
		{filepath.Join(root, "kubepods"), false, ""},
		{filepath.Join(root, "kubelet", "kubepods"), false, "kubelet"},
	}
	for _, c := range candidates {
		if _, err := os.Stat(c.path); err == nil {
			l.KubepodsPrefix = c.prefix
			if c.systemd {
				l.Systemd = true
				l.SystemdScopePrefix = findSystemdScopePrefixOnDisk(c.path)
			}
			l.Valid = true
			return
		}
	}
}

// findSystemdScopePrefixOnDisk descends into a systemd kubepods root and
// returns the scope prefix of the first container scope directory it finds.
// Returns "" if no scope is present (e.g. no pods scheduled yet).
func findSystemdScopePrefixOnDisk(kubepodsRoot string) string {
	var prefix string
	_ = filepath.WalkDir(kubepodsRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && strings.HasSuffix(d.Name(), ".scope") {
			if sp := extractSystemdScopePrefix(path); sp != "" {
				prefix = sp
				return filepath.SkipAll
			}
		}
		return nil
	})
	return prefix
}

// extractSystemdScopePrefix returns the systemd scope prefix from a path
// whose final segment is "<prefix>-<id>.scope" (e.g.
// "cri-containerd-<id>.scope" → "cri-containerd"). Returns "" if the leaf
// isn't a recognizable systemd scope.
func extractSystemdScopePrefix(p string) string {
	leaf := filepath.Base(strings.TrimSuffix(p, "/"))
	body, ok := strings.CutSuffix(leaf, ".scope")
	if !ok {
		return ""
	}
	idx := strings.LastIndex(body, "-")
	if idx <= 0 {
		return ""
	}
	return body[:idx]
}

var kubepodsSliceRe = regexp.MustCompile(`^([a-z0-9_]+-)?kubepods\.slice$`)

func extractKubepodsPrefix(cgroupPath string) (prefix string, ok bool) {
	segs := strings.Split(strings.Trim(cgroupPath, "/"), "/")
	for i, seg := range segs {
		if seg == "" {
			continue
		}
		// systemd: "kubepods.slice" or "<prefix>-kubepods.slice"
		if m := kubepodsSliceRe.FindStringSubmatch(seg); m != nil {
			return strings.TrimSuffix(m[1], "-"), true
		}
		// cgroupfs: bare "kubepods" directory; whatever precedes it is
		if seg == "kubepods" {
			return filepath.Join(segs[:i]...), true
		}
	}
	return "", false
}

// containerCgroupDir returns the absolute on-host directory holding the
// container's cgroup.procs file.
func containerCgroupDir(l cgroupLayout, pc PodContainer) (string, error) {
	i := strings.Index(pc.ContainerID, "://")
	if i < 0 {
		return "", fmt.Errorf("invalid container ref: %q", pc.ContainerID)
	}
	id := pc.ContainerID[i+3:]
	pod, err := podCgroupDir(l, pc.QOSClass, pc.PodUID)
	if err != nil {
		return "", err
	}
	if l.Systemd {
		return filepath.Join(pod, containerSystemdScope(l.SystemdScopePrefix, id)), nil
	}
	return filepath.Join(pod, id), nil
}

func podCgroupDir(l cgroupLayout, qos, podUID string) (string, error) {
	root := l.root
	if l.Systemd {
		slice := podSystemdSlice(l.KubepodsPrefix, qos, podUID)
		tree, err := systemd.ExpandSlice(slice)
		if err != nil {
			return "", fmt.Errorf("expand slice %q: %w", slice, err)
		}
		return filepath.Join(root, tree), nil
	}
	return filepath.Join(root, cgroupfsPodRel(l.KubepodsPrefix, qos, podUID)), nil
}

// podSystemdSlice builds the systemd slice name for a pod. Per kubelet,
// the pod UID has '-' escaped to '_' inside the slice segment.
func podSystemdSlice(kubepodsPrefix, qos, podUID string) string {
	uid := strings.ReplaceAll(podUID, "-", "_")
	parts := []string{}
	if kubepodsPrefix != "" {
		parts = append(parts, kubepodsPrefix)
	}
	parts = append(parts, "kubepods")
	if q := strings.ToLower(qos); q != "" && q != "guaranteed" {
		parts = append(parts, q)
	}
	parts = append(parts, "pod"+uid)
	return strings.Join(parts, "-") + ".slice"
}

func cgroupfsPodRel(prefix, qos, podUID string) string {
	base := filepath.Join(prefix, "kubepods")
	if q := strings.ToLower(qos); q != "" && q != "guaranteed" {
		base = filepath.Join(base, q)
	}
	return filepath.Join(base, "pod"+podUID)
}

func containerSystemdScope(scopePrefix, id string) string {
	return fmt.Sprintf("%s-%s.scope", scopePrefix, id)
}

// ErrCgroupMissing is returned when the resolved cgroup directory does
// not exist on disk. Callers can treat this as "container already gone".
var ErrCgroupMissing = errors.New("container cgroup not found")

// pidsInContainer returns the PIDs currently in the container's cgroup.
// It returns ErrCgroupMissing if the resolved cgroup directory does not
// exist (the container exited or never started).
func pidsInContainer(l cgroupLayout, pc PodContainer) ([]int, error) {
	dir, err := containerCgroupDir(l, pc)
	if err != nil {
		return nil, err
	}
	pids, err := cgroups.GetPids(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrCgroupMissing
		}
		return nil, fmt.Errorf("read cgroup.procs %q: %w", dir, err)
	}
	return pids, nil
}
