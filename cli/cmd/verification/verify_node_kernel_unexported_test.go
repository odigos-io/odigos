package verification

import (
	"testing"

	"github.com/moby/moby/pkg/parsers/kernel"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestVerifyNodeKernel(t *testing.T) {
	minimumKernelVersion := mustParseNodeKernelVersion(t)

	tt := []struct {
		_             struct{}
		Name          string
		NodeItem      []corev1.Node
		ExpectedError error
	}{
		{
			Name:          "empty node",
			NodeItem:      []corev1.Node{},
			ExpectedError: ErrOdigosUnsupportedKernel,
		},
		{
			Name: "all unparseable kernel version",
			NodeItem: []corev1.Node{
				createDummyNodeWithKernelVersion(t, "abcde"),
			},
			ExpectedError: ErrOdigosUnsupportedKernel,
		},
		{
			Name: "one unparseable kernel version",
			NodeItem: []corev1.Node{
				createDummyNodeWithKernelVersion(t, "abcde"),
				createDummyNodeWithKernelVersion(t, "4.15"),
			},
			ExpectedError: nil,
		},
		{
			Name: "all minimum kernel version",
			NodeItem: []corev1.Node{
				createDummyNodeWithKernelVersion(t, "4.14"),
				createDummyNodeWithKernelVersion(t, "4.14"),
			},
			ExpectedError: nil,
		},
		{
			Name: "one under minimum kernel version",
			NodeItem: []corev1.Node{
				createDummyNodeWithKernelVersion(t, "4.13"),
				createDummyNodeWithKernelVersion(t, "4.14"),
			},
			ExpectedError: nil,
		},
		{
			Name: "all under minimum kernel version",
			NodeItem: []corev1.Node{
				createDummyNodeWithKernelVersion(t, "4.13"),
				createDummyNodeWithKernelVersion(t, "4.13"),
			},
			ExpectedError: ErrOdigosUnsupportedKernel,
		},
	}

	for i := range tt {
		tc := tt[i]

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)

			actualErr := verifyNodeKernel(minimumKernelVersion, tc.NodeItem)
			if tc.ExpectedError != nil {
				r.Error(actualErr)
				r.ErrorIs(tc.ExpectedError, actualErr)
			} else {
				r.NoError(actualErr)
			}
		})
	}
}

func mustParseNodeKernelVersion(t *testing.T) *kernel.VersionInfo {
	t.Helper()

	v, err := kernel.ParseRelease(OdigosMinimumKernelVersion)
	if err != nil {
		panic(err.Error())
	}

	return v
}

func createDummyNodeWithKernelVersion(t *testing.T, kernelVersion string) corev1.Node {
	t.Helper()
	return corev1.Node{
		Status: corev1.NodeStatus{
			NodeInfo: corev1.NodeSystemInfo{
				KernelVersion: kernelVersion,
			},
		},
	}
}
