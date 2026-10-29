package process

import "testing"

func TestPodContainerOfMountRoot(t *testing.T) {
	tests := []struct {
		name string
		root string
		want PodContainerKey
		ok   bool
	}{
		{
			name: "termination log",
			root: "/var/lib/kubelet/pods/3d75f212-95fe-4403-9552-13f01467308e/containers/rates-service/c430d5be",
			want: PodContainerKey{PodUID: "3d75f212-95fe-4403-9552-13f01467308e", ContainerName: "rates-service"},
			ok:   true,
		},
		{
			name: "kubelet root on a docker volume (kind)",
			root: "/docker/volumes/aa7e/_data/lib/kubelet/pods/3d75f212-95fe-4403-9552-13f01467308e/containers/rates-service/c430d5be",
			want: PodContainerKey{PodUID: "3d75f212-95fe-4403-9552-13f01467308e", ContainerName: "rates-service"},
			ok:   true,
		},
		{name: "etc-hosts", root: "/var/lib/kubelet/pods/3d75f212-95fe-4403-9552-13f01467308e/etc-hosts"},
		{name: "volume", root: "/var/lib/kubelet/pods/3d75f212/volumes/kubernetes.io~configmap/ca"},
		{name: "no container id", root: "/var/lib/kubelet/pods/3d75f212/containers/rates-service"},
		{name: "not kubelet", root: "/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := podContainerOfMountRoot(tt.root)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("podContainerOfMountRoot(%q) = %+v, %v; want %+v, %v", tt.root, got, ok, tt.want, tt.ok)
			}
		})
	}
}
