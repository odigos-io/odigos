package process

import (
	"testing"
	corev1 "k8s.io/api/core/v1"
)

func TestExtractKubepodsPrefix(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"stock systemd v2 burstable", "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod123.slice/foo.scope", "", true},
		{"stock systemd v2 besteffort", "/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod123.slice/foo.scope", "", true},
		{"stock systemd v2 guaranteed", "/kubepods.slice/kubepods-pod123.slice/foo.scope", "", true},
		{"kind systemd v2 burstable", "/kubelet.slice/kubelet-kubepods.slice/kubelet-kubepods-burstable.slice/kubelet-kubepods-burstable-pod123.slice/foo.scope", "kubelet", true},
		{"cgroupfs v2 burstable", "/kubepods/burstable/pod123/abc", "", true},
		{"cgroupfs v2 besteffort", "/kubepods/besteffort/pod123/abc", "", true},
		{"cgroupfs v2 guaranteed", "/kubepods/pod123/abc", "", true},
		{"non-kube", "/system.slice/sshd.service", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractKubepodsPrefix(tt.in)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("got (%q,%v) want (%q,%v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestExtractSystemdScopePrefix(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"containerd scope", "/kubepods.slice/kubepods-pod123.slice/cri-containerd-abc.scope", "cri-containerd"},
		{"docker scope", "/kubepods.slice/kubepods-pod123.slice/docker-abc.scope", "docker"},
		{"crio scope", "/kubepods.slice/kubepods-pod123.slice/crio-abc.scope", "crio"},
		{"leaf only", "cri-containerd-abc.scope", "cri-containerd"},
		{"trailing slash", "/kubepods.slice/kubepods-pod123.slice/cri-containerd-abc.scope/", "cri-containerd"},
		{"not a scope", "/kubepods.slice/kubepods-pod123.slice", ""},
		{"no dash", "abc.scope", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSystemdScopePrefix(tt.in)
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestContainerCgroupDir(t *testing.T) {
	const (
		uid      = "12345678-1234-1234-1234-123456789012"
		uidUnder = "12345678_1234_1234_1234_123456789012"
	)

	stockV2 := cgroupLayout{Systemd: true, SystemdScopePrefix: "cri-containerd", root: "/sys/fs/cgroup"}
	kindV2 := cgroupLayout{Systemd: true, SystemdScopePrefix: "cri-containerd", KubepodsPrefix: "kubelet", root: "/sys/fs/cgroup"}
	cgfsV2 := cgroupLayout{Systemd: false, root: "/sys/fs/cgroup"}

	cid := "containerd://abc"
	scope := "cri-containerd-abc.scope"

	tests := []struct {
		name   string
		layout cgroupLayout
		qos    corev1.PodQOSClass
		want   string
	}{
		{
			"stock systemd v2 guaranteed", stockV2, corev1.PodQOSGuaranteed,
			"/sys/fs/cgroup/kubepods.slice/kubepods-pod" + uidUnder + ".slice/" + scope,
		},
		{
			"stock systemd v2 burstable", stockV2, corev1.PodQOSBurstable,
			"/sys/fs/cgroup/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod" + uidUnder + ".slice/" + scope,
		},
		{
			"stock systemd v2 besteffort", stockV2, corev1.PodQOSBestEffort,
			"/sys/fs/cgroup/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod" + uidUnder + ".slice/" + scope,
		},

		{
			"kind systemd v2 guaranteed", kindV2, corev1.PodQOSGuaranteed,
			"/sys/fs/cgroup/kubelet.slice/kubelet-kubepods.slice/kubelet-kubepods-pod" + uidUnder + ".slice/" + scope,
		},
		{
			"kind systemd v2 burstable", kindV2, corev1.PodQOSBurstable,
			"/sys/fs/cgroup/kubelet.slice/kubelet-kubepods.slice/kubelet-kubepods-burstable.slice/kubelet-kubepods-burstable-pod" + uidUnder + ".slice/" + scope,
		},
		{
			"kind systemd v2 besteffort", kindV2, corev1.PodQOSBestEffort,
			"/sys/fs/cgroup/kubelet.slice/kubelet-kubepods.slice/kubelet-kubepods-besteffort.slice/kubelet-kubepods-besteffort-pod" + uidUnder + ".slice/" + scope,
		},

		{
			"cgroupfs v2 guaranteed", cgfsV2, corev1.PodQOSGuaranteed,
			"/sys/fs/cgroup/kubepods/pod" + uid + "/abc",
		},
		{
			"cgroupfs v2 burstable", cgfsV2, corev1.PodQOSBurstable,
			"/sys/fs/cgroup/kubepods/burstable/pod" + uid + "/abc",
		},
		{
			"cgroupfs v2 besteffort", cgfsV2, corev1.PodQOSBestEffort,
			"/sys/fs/cgroup/kubepods/besteffort/pod" + uid + "/abc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := PodContainer{
				PodContainerKey: PodContainerKey{PodUID: uid, ContainerName: "c"},
				QOSClass:        tt.qos,
				ContainerID:     cid,
			}
			got, err := containerCgroupDir(tt.layout, pc)
			if err != nil {
				t.Fatalf("ContainerCgroupDir: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
