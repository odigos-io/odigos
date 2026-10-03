package workload_test

import (
	"testing"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/tj/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	configSourceAnnotation = "kubernetes.io/config.source"
	configHashAnnotation   = "kubernetes.io/config.hash"
)

// staticPodPod builds a pod owned by a Node and marked as coming from a static
// manifest, which is how the kubelet reports static pods.
func staticPodPod(name string, nodeName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       "kube-system",
			OwnerReferences: []metav1.OwnerReference{{Kind: "Node", Name: nodeName}},
			Annotations:     map[string]string{configSourceAnnotation: "file"},
		},
		Spec: corev1.PodSpec{NodeName: nodeName},
	}
}

func TestIsStaticPod(t *testing.T) {
	tests := []struct {
		name        string
		ownerRefs   []metav1.OwnerReference
		annotations map[string]string
		want        bool
	}{
		{
			name:        "node owner with a file config source",
			ownerRefs:   []metav1.OwnerReference{{Kind: "Node", Name: "node-1"}},
			annotations: map[string]string{configSourceAnnotation: "file"},
			want:        true,
		},
		{
			name:        "node owner with an http config source",
			ownerRefs:   []metav1.OwnerReference{{Kind: "Node", Name: "node-1"}},
			annotations: map[string]string{configSourceAnnotation: "http"},
			want:        true,
		},
		{
			// a pod created through the api server and scheduled to the node is
			// not static, even though the kubelet also owns it.
			name:        "node owner with an api config source",
			ownerRefs:   []metav1.OwnerReference{{Kind: "Node", Name: "node-1"}},
			annotations: map[string]string{configSourceAnnotation: "api"},
			want:        false,
		},
		{
			name:        "node owner with no annotations at all",
			ownerRefs:   []metav1.OwnerReference{{Kind: "Node", Name: "node-1"}},
			annotations: nil,
			want:        false,
		},
		{
			name:        "node owner with annotations but no config source",
			ownerRefs:   []metav1.OwnerReference{{Kind: "Node", Name: "node-1"}},
			annotations: map[string]string{"other": "value"},
			want:        false,
		},
		{
			name:        "node owner with an empty config source",
			ownerRefs:   []metav1.OwnerReference{{Kind: "Node", Name: "node-1"}},
			annotations: map[string]string{configSourceAnnotation: ""},
			want:        false,
		},
		{
			name:        "config source is matched case sensitively",
			ownerRefs:   []metav1.OwnerReference{{Kind: "Node", Name: "node-1"}},
			annotations: map[string]string{configSourceAnnotation: "File"},
			want:        false,
		},
		{
			name:        "no owner references",
			ownerRefs:   nil,
			annotations: map[string]string{configSourceAnnotation: "file"},
			want:        false,
		},
		{
			// the file config source alone is not enough, the pod must be owned by a node.
			name:        "owned by a replicaset",
			ownerRefs:   []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "rs-1"}},
			annotations: map[string]string{configSourceAnnotation: "file"},
			want:        false,
		},
		{
			// the node owner may appear anywhere in the list.
			name: "node owner listed after another owner",
			ownerRefs: []metav1.OwnerReference{
				{Kind: "ReplicaSet", Name: "rs-1"},
				{Kind: "Node", Name: "node-1"},
			},
			annotations: map[string]string{configSourceAnnotation: "file"},
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "kube-apiserver-node-1",
					Namespace:       "kube-system",
					OwnerReferences: tt.ownerRefs,
					Annotations:     tt.annotations,
				},
			}
			assert.Equal(t, tt.want, workload.IsStaticPod(pod))
		})
	}
}

// A static pod's real UID is published in the config.hash annotation; the object
// UID is a kubelet-mirrored value that odiglet cannot correlate to a process.
func TestPodUID(t *testing.T) {
	t.Run("static pod uses the config hash annotation", func(t *testing.T) {
		pod := staticPodPod("kube-apiserver-node-1", "node-1")
		pod.UID = "object-uid"
		pod.Annotations[configHashAnnotation] = "config-hash-uid"

		assert.Equal(t, "config-hash-uid", workload.PodUID(pod))
	})

	t.Run("static pod with no config hash annotation", func(t *testing.T) {
		pod := staticPodPod("kube-apiserver-node-1", "node-1")
		pod.UID = "object-uid"

		assert.Equal(t, "", workload.PodUID(pod))
	})

	t.Run("regular pod uses the object uid", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "app-pod", Namespace: "default", UID: "object-uid"},
		}

		assert.Equal(t, "object-uid", workload.PodUID(pod))
	})

	t.Run("regular pod with a config hash annotation still uses the object uid", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "app-pod",
				Namespace:   "default",
				UID:         "object-uid",
				Annotations: map[string]string{configHashAnnotation: "config-hash-uid"},
			},
		}

		assert.Equal(t, "object-uid", workload.PodUID(pod))
	})
}

// Static pod names are suffixed with the node name by the kubelet, and odigos
// needs the manifest name back to correlate the pod with its Source.
func TestStaticPodName(t *testing.T) {
	t.Run("strips the node name suffix", func(t *testing.T) {
		pod := staticPodPod("kube-apiserver-node-1", "node-1")
		assert.Equal(t, "kube-apiserver", workload.StaticPodName(pod))
	})

	t.Run("node name appearing mid-name is not stripped", func(t *testing.T) {
		pod := staticPodPod("kube-apiserver-node-1-replica", "node-1")
		assert.Equal(t, "", workload.StaticPodName(pod))
	})

	t.Run("name without the node name suffix", func(t *testing.T) {
		pod := staticPodPod("kube-apiserver", "node-1")
		assert.Equal(t, "", workload.StaticPodName(pod))
	})

	t.Run("static pod not assigned to a node", func(t *testing.T) {
		pod := staticPodPod("kube-apiserver-node-1", "node-1")
		pod.Spec.NodeName = ""
		assert.Equal(t, "", workload.StaticPodName(pod))
	})

	t.Run("pod that is not static", func(t *testing.T) {
		pod := staticPodPod("kube-apiserver-node-1", "node-1")
		pod.Annotations[configSourceAnnotation] = "api"
		assert.Equal(t, "", workload.StaticPodName(pod))
	})

	t.Run("nil pod", func(t *testing.T) {
		assert.Equal(t, "", workload.StaticPodName(nil))
	})
}
