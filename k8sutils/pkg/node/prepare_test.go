package node

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
)

const prepareTestNodeName = "gke-pool-1-abcd1234"

// nodeAPIFailure is the response the stub API server returns instead of the node.
type nodeAPIFailure struct {
	code   int
	reason metav1.StatusReason
}

// nodeAPIStub serves the read and write of a single node. PrepareNodeForOdigosInstallation
// takes a *kubernetes.Clientset rather than kubernetes.Interface, so a fake clientset cannot
// be substituted; a real clientset pointed at this handler exercises the same code path,
// including the serialization and the API error mapping that drive the conflict retry.
type nodeAPIStub struct {
	mu sync.Mutex

	node *corev1.Node

	// getFailure, when set, is returned for every read.
	getFailure *nodeAPIFailure
	// updateFailures is consumed one entry per write; a nil entry accepts the write.
	updateFailures []*nodeAPIFailure
	// onUpdateRejected runs after a write was rejected, to let a test simulate another
	// writer changing the node before the retry re-reads it.
	onUpdateRejected func(node *corev1.Node)

	gets    int
	updates int
	// submitted records every node object the client sent, in order.
	submitted []*corev1.Node
}

func (s *nodeAPIStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r.URL.Path != "/api/v1/nodes/"+s.node.Name {
		writeNodeAPIFailure(w, &nodeAPIFailure{code: http.StatusNotFound, reason: metav1.StatusReasonNotFound})
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.gets++
		if s.getFailure != nil {
			writeNodeAPIFailure(w, s.getFailure)
			return
		}
		writeNodeAPINode(w, s.node)

	case http.MethodPut:
		s.updates++

		var sent corev1.Node
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			writeNodeAPIFailure(w, &nodeAPIFailure{code: http.StatusBadRequest, reason: metav1.StatusReasonBadRequest})
			return
		}
		s.submitted = append(s.submitted, &sent)

		var failure *nodeAPIFailure
		if len(s.updateFailures) > 0 {
			failure = s.updateFailures[0]
			s.updateFailures = s.updateFailures[1:]
		}
		if failure != nil {
			if s.onUpdateRejected != nil {
				s.onUpdateRejected(s.node)
			}
			writeNodeAPIFailure(w, failure)
			return
		}

		s.node = &sent
		writeNodeAPINode(w, &sent)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *nodeAPIStub) counts() (gets int, updates int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gets, s.updates
}

// lastSubmitted returns the node object of the final write the client issued.
func (s *nodeAPIStub) lastSubmitted(t *testing.T) *corev1.Node {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()
	require.NotEmpty(t, s.submitted, "no node update was sent")

	return s.submitted[len(s.submitted)-1]
}

func writeNodeAPINode(w http.ResponseWriter, node *corev1.Node) {
	out := node.DeepCopy()
	out.TypeMeta = metav1.TypeMeta{Kind: "Node", APIVersion: "v1"}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}

func writeNodeAPIFailure(w http.ResponseWriter, failure *nodeAPIFailure) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(failure.code)
	_ = json.NewEncoder(w).Encode(&metav1.Status{
		TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
		Status:   metav1.StatusFailure,
		Code:     int32(failure.code),
		Reason:   failure.reason,
		Message:  string(failure.reason),
	})
}

func newNodeAPIClientset(t *testing.T, stub *nodeAPIStub) *kubernetes.Clientset {
	t.Helper()

	server := httptest.NewServer(stub)
	t.Cleanup(server.Close)

	// client-go talks protobuf to the built-in types by default; JSON keeps the stub readable.
	clientset, err := kubernetes.NewForConfig(&rest.Config{
		Host:          server.URL,
		ContentConfig: rest.ContentConfig{ContentType: "application/json"},
	})
	require.NoError(t, err)

	return clientset
}

// prepareTestNode is a node as Karpenter provisions it for Odigos: still carrying the startup
// taint that keeps workloads off it until odiglet has finished initializing.
func prepareTestNode() *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   prepareTestNodeName,
			Labels: map[string]string{"kubernetes.io/os": "linux"},
		},
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{
				{Key: "dedicated", Value: "gpu", Effect: corev1.TaintEffectNoSchedule},
				{Key: consts.KarpenterStartupTaintKey, Effect: corev1.TaintEffectNoSchedule},
				{Key: "node.kubernetes.io/unreachable", Effect: corev1.TaintEffectNoExecute},
			},
		},
	}
}

func TestPrepareNodeForOdigosInstallationRemovesTheStartupTaint(t *testing.T) {
	t.Setenv(consts.OdigosTierEnvVarName, string(common.CommunityOdigosTier))

	stub := &nodeAPIStub{node: prepareTestNode()}
	require.NoError(t, PrepareNodeForOdigosInstallation(newNodeAPIClientset(t, stub), prepareTestNodeName))

	// Nothing schedules onto a Karpenter node while the startup taint is present, so it must
	// be dropped, and every other taint must survive in order.
	assert.Equal(t, []corev1.Taint{
		{Key: "dedicated", Value: "gpu", Effect: corev1.TaintEffectNoSchedule},
		{Key: "node.kubernetes.io/unreachable", Effect: corev1.TaintEffectNoExecute},
	}, stub.lastSubmitted(t).Spec.Taints)
}

func TestPrepareNodeForOdigosInstallationOnlyRemovesTheExactStartupTaint(t *testing.T) {
	tests := []struct {
		name    string
		taint   corev1.Taint
		removed bool
	}{
		{
			name:    "the Odigos startup taint",
			taint:   corev1.Taint{Key: consts.KarpenterStartupTaintKey, Effect: corev1.TaintEffectNoSchedule},
			removed: true,
		},
		{
			name:    "the startup taint key with another effect",
			taint:   corev1.Taint{Key: consts.KarpenterStartupTaintKey, Effect: corev1.TaintEffectNoExecute},
			removed: false,
		},
		{
			name:    "another key with the same effect",
			taint:   corev1.Taint{Key: "odigos.io/other", Effect: corev1.TaintEffectNoSchedule},
			removed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(consts.OdigosTierEnvVarName, string(common.CommunityOdigosTier))

			node := prepareTestNode()
			node.Spec.Taints = []corev1.Taint{tt.taint}
			stub := &nodeAPIStub{node: node}

			require.NoError(t, PrepareNodeForOdigosInstallation(newNodeAPIClientset(t, stub), prepareTestNodeName))

			if tt.removed {
				assert.Empty(t, stub.lastSubmitted(t).Spec.Taints)
			} else {
				assert.Equal(t, []corev1.Taint{tt.taint}, stub.lastSubmitted(t).Spec.Taints)
			}
		})
	}
}

func TestPrepareNodeForOdigosInstallationLabelsTheNodeByTier(t *testing.T) {
	tests := []struct {
		name           string
		tier           string
		existingLabels map[string]string
		wantLabels     map[string]string
	}{
		{
			name:           "community install",
			tier:           string(common.CommunityOdigosTier),
			existingLabels: map[string]string{"kubernetes.io/os": "linux"},
			wantLabels: map[string]string{
				"kubernetes.io/os":                 "linux",
				k8sconsts.OdigletOSSInstalledLabel: k8sconsts.OdigletInstalledLabelValue,
			},
		},
		{
			name:           "enterprise install",
			tier:           string(common.OnPremOdigosTier),
			existingLabels: map[string]string{"kubernetes.io/os": "linux"},
			wantLabels: map[string]string{
				"kubernetes.io/os":                        "linux",
				k8sconsts.OdigletEnterpriseInstalledLabel: k8sconsts.OdigletInstalledLabelValue,
			},
		},
		{
			name: "a node with no labels at all",
			tier: string(common.CommunityOdigosTier),
			wantLabels: map[string]string{
				k8sconsts.OdigletOSSInstalledLabel: k8sconsts.OdigletInstalledLabelValue,
			},
		},
		{
			// Upgrading to enterprise must clear the community label, otherwise the node ends
			// up selected by both tiers' node affinity at once.
			name:           "upgrade from community to enterprise",
			tier:           string(common.OnPremOdigosTier),
			existingLabels: map[string]string{k8sconsts.OdigletOSSInstalledLabel: k8sconsts.OdigletInstalledLabelValue},
			wantLabels: map[string]string{
				k8sconsts.OdigletEnterpriseInstalledLabel: k8sconsts.OdigletInstalledLabelValue,
			},
		},
		{
			name:           "downgrade from enterprise to community",
			tier:           string(common.CommunityOdigosTier),
			existingLabels: map[string]string{k8sconsts.OdigletEnterpriseInstalledLabel: k8sconsts.OdigletInstalledLabelValue},
			wantLabels: map[string]string{
				k8sconsts.OdigletOSSInstalledLabel: k8sconsts.OdigletInstalledLabelValue,
			},
		},
		{
			name:           "the cloud tier is labelled as community and clears the enterprise label",
			tier:           string(common.CloudOdigosTier),
			existingLabels: map[string]string{k8sconsts.OdigletEnterpriseInstalledLabel: k8sconsts.OdigletInstalledLabelValue},
			wantLabels: map[string]string{
				k8sconsts.OdigletOSSInstalledLabel: k8sconsts.OdigletInstalledLabelValue,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(consts.OdigosTierEnvVarName, tt.tier)

			node := prepareTestNode()
			node.Labels = tt.existingLabels
			stub := &nodeAPIStub{node: node}

			require.NoError(t, PrepareNodeForOdigosInstallation(newNodeAPIClientset(t, stub), prepareTestNodeName))

			assert.Equal(t, tt.wantLabels, stub.lastSubmitted(t).Labels)
		})
	}
}

func TestPrepareNodeForOdigosInstallationWhenTheNodeCannotBeRead(t *testing.T) {
	t.Setenv(consts.OdigosTierEnvVarName, string(common.CommunityOdigosTier))

	stub := &nodeAPIStub{
		node:       prepareTestNode(),
		getFailure: &nodeAPIFailure{code: http.StatusForbidden, reason: metav1.StatusReasonForbidden},
	}

	err := PrepareNodeForOdigosInstallation(newNodeAPIClientset(t, stub), prepareTestNodeName)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get node "+prepareTestNodeName)
	// The API error is wrapped rather than replaced, so callers can still classify it.
	assert.True(t, apierrors.IsForbidden(err))

	_, updates := stub.counts()
	assert.Zero(t, updates)
}

func TestPrepareNodeForOdigosInstallationRetriesAConflictAgainstAFreshRead(t *testing.T) {
	t.Setenv(consts.OdigosTierEnvVarName, string(common.CommunityOdigosTier))

	stub := &nodeAPIStub{
		node: prepareTestNode(),
		updateFailures: []*nodeAPIFailure{
			{code: http.StatusConflict, reason: metav1.StatusReasonConflict},
			nil,
		},
	}
	// Another controller taints the node while the first write is in flight; the retry has to
	// re-read the node, or that taint is silently reverted.
	stub.onUpdateRejected = func(node *corev1.Node) {
		node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
			Key:    "node.kubernetes.io/disk-pressure",
			Effect: corev1.TaintEffectNoSchedule,
		})
	}

	require.NoError(t, PrepareNodeForOdigosInstallation(newNodeAPIClientset(t, stub), prepareTestNodeName))

	gets, updates := stub.counts()
	assert.Equal(t, 2, gets)
	assert.Equal(t, 2, updates)

	submitted := stub.lastSubmitted(t)
	assert.Equal(t, []corev1.Taint{
		{Key: "dedicated", Value: "gpu", Effect: corev1.TaintEffectNoSchedule},
		{Key: "node.kubernetes.io/unreachable", Effect: corev1.TaintEffectNoExecute},
		{Key: "node.kubernetes.io/disk-pressure", Effect: corev1.TaintEffectNoSchedule},
	}, submitted.Spec.Taints)
	assert.Equal(t, k8sconsts.OdigletInstalledLabelValue, submitted.Labels[k8sconsts.OdigletOSSInstalledLabel])
}

func TestPrepareNodeForOdigosInstallationDoesNotRetryOtherWriteErrors(t *testing.T) {
	t.Setenv(consts.OdigosTierEnvVarName, string(common.CommunityOdigosTier))

	stub := &nodeAPIStub{
		node:           prepareTestNode(),
		updateFailures: []*nodeAPIFailure{{code: http.StatusForbidden, reason: metav1.StatusReasonForbidden}},
	}

	err := PrepareNodeForOdigosInstallation(newNodeAPIClientset(t, stub), prepareTestNodeName)

	require.Error(t, err)
	assert.True(t, apierrors.IsForbidden(err))
	// Only a conflict is worth retrying; a permission problem must surface immediately so
	// odiglet fails its init rather than spinning on the API server.
	gets, updates := stub.counts()
	assert.Equal(t, 1, gets)
	assert.Equal(t, 1, updates)
}

func TestPrepareNodeForOdigosInstallationGivesUpOnRepeatedConflicts(t *testing.T) {
	t.Setenv(consts.OdigosTierEnvVarName, string(common.CommunityOdigosTier))

	conflicts := make([]*nodeAPIFailure, 0, 10)
	for range 10 {
		conflicts = append(conflicts, &nodeAPIFailure{code: http.StatusConflict, reason: metav1.StatusReasonConflict})
	}
	stub := &nodeAPIStub{node: prepareTestNode(), updateFailures: conflicts}

	err := PrepareNodeForOdigosInstallation(newNodeAPIClientset(t, stub), prepareTestNodeName)

	require.Error(t, err)
	assert.True(t, apierrors.IsConflict(err))
	// It retries a bounded number of times and then reports the conflict, rather than looping.
	_, updates := stub.counts()
	assert.Greater(t, updates, 1)
	assert.Less(t, updates, len(conflicts))
}
