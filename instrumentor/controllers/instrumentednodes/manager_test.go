package instrumentednodes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// ****************
// Manager stubs
// ****************

type recordedIndex struct {
	object       client.Object
	field        string
	extractValue client.IndexerFunc
}

type stubFieldIndexer struct {
	indexes []recordedIndex
	err     error
}

func (s *stubFieldIndexer) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	if s.err != nil {
		return s.err
	}
	s.indexes = append(s.indexes, recordedIndex{object: obj, field: field, extractValue: extractValue})
	return nil
}

// stubManager implements only the parts of ctrl.Manager the controller builders
// touch at setup time; the embedded interface panics on anything else, which
// keeps the stub honest about what SetupWithManager is allowed to use.
type stubManager struct {
	ctrl.Manager
	indexer   *stubFieldIndexer
	client    client.Client
	scheme    *runtime.Scheme
	runnables []manager.Runnable
}

func newStubManager(scheme *runtime.Scheme) *stubManager {
	return &stubManager{
		indexer: &stubFieldIndexer{},
		client:  fake.NewClientBuilder().WithScheme(scheme).Build(),
		scheme:  scheme,
	}
}

func (m *stubManager) GetFieldIndexer() client.FieldIndexer { return m.indexer }
func (m *stubManager) GetClient() client.Client             { return m.client }
func (m *stubManager) GetScheme() *runtime.Scheme           { return m.scheme }
func (m *stubManager) GetLogger() logr.Logger               { return logr.Discard() }
func (m *stubManager) GetCache() cache.Cache                { return nil }

func (m *stubManager) GetControllerOptions() config.Controller {
	// controller names are registered process-wide, so skipping the check keeps
	// tests independent of each other and of their execution order.
	skipNameValidation := true
	return config.Controller{SkipNameValidation: &skipNameValidation}
}

func (m *stubManager) Add(runnable manager.Runnable) error {
	m.runnables = append(m.runnables, runnable)
	return nil
}

// registeredPodNodeNameIndex runs the production setup and returns the pod
// index function it registered.
func registeredPodNodeNameIndex(t *testing.T) client.IndexerFunc {
	t.Helper()
	mgr := newStubManager(instrumentedNodesScheme())
	require.NoError(t, SetupWithManager(context.Background(), mgr, time.Minute))
	require.Len(t, mgr.indexer.indexes, 1)
	return mgr.indexer.indexes[0].extractValue
}

// ****************
// SetupWithManager() tests
// ****************

func TestSetupWithManager_IndexesPodsByNodeNameAndStartsAllControllers(t *testing.T) {
	mgr := newStubManager(instrumentedNodesScheme())

	require.NoError(t, SetupWithManager(context.Background(), mgr, time.Minute))

	require.Len(t, mgr.indexer.indexes, 1)
	assert.IsType(t, &corev1.Pod{}, mgr.indexer.indexes[0].object)
	assert.Equal(t, "spec.nodeName", mgr.indexer.indexes[0].field)
	// one controller each for pods, nodes and instrumentation configs
	assert.Len(t, mgr.runnables, 3)
}

func TestSetupWithManager_PodNodeNameIndex(t *testing.T) {
	index := registeredPodNodeNameIndex(t)

	tests := []struct {
		name     string
		object   client.Object
		expected []string
	}{
		{
			name:     "a scheduled pod is indexed by its node",
			object:   podOnNode("scheduled", agentsNodeName),
			expected: []string{agentsNodeName},
		},
		{
			name:     "a pending pod has no node to index by",
			object:   podOnNode("pending", ""),
			expected: nil,
		},
		{
			name:     "an object that is not a pod is not indexed",
			object:   nodeNamed(agentsNodeName),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, index(tt.object))
		})
	}
}

func TestSetupWithManager_IndexErrorIsPropagated(t *testing.T) {
	mgr := newStubManager(instrumentedNodesScheme())
	mgr.indexer.err = errors.New("index already exists")

	err := SetupWithManager(context.Background(), mgr, time.Minute)

	assert.ErrorContains(t, err, "index already exists")
	assert.Empty(t, mgr.runnables)
}

func TestSetupWithManager_ControllerBuildErrorIsPropagated(t *testing.T) {
	nodeOnlyScheme := runtime.NewScheme()
	nodeOnlyScheme.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Node{}, &corev1.NodeList{})
	metav1.AddToGroupVersion(nodeOnlyScheme, corev1.SchemeGroupVersion)

	podOnlyScheme := runtime.NewScheme()
	podOnlyScheme.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Pod{}, &corev1.PodList{})
	metav1.AddToGroupVersion(podOnlyScheme, corev1.SchemeGroupVersion)

	withoutOdigosTypes := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(withoutOdigosTypes))

	tests := []struct {
		name              string
		scheme            *runtime.Scheme
		controllersBefore int
	}{
		{
			name:              "the pods controller cannot be built",
			scheme:            nodeOnlyScheme,
			controllersBefore: 0,
		},
		{
			name:              "the nodes controller cannot be built",
			scheme:            podOnlyScheme,
			controllersBefore: 1,
		},
		{
			name:              "the instrumentation config controller cannot be built",
			scheme:            withoutOdigosTypes,
			controllersBefore: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := newStubManager(tt.scheme)

			err := SetupWithManager(context.Background(), mgr, time.Minute)

			assert.Error(t, err)
			assert.Len(t, mgr.runnables, tt.controllersBefore)
		})
	}
}
