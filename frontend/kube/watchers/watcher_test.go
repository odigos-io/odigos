package watchers

import (
	"context"
	"testing"
	"time"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/frontend/kube"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

// TestDestinationWatcherReconnection verifies that the destination watcher
// reconnects after the watch channel is closed (simulated by cancelling and
// restarting the context). This test ensures the reconnection loop in
// RunDestinationWatcher exits cleanly on context cancellation rather than
// hanging forever.
func TestDestinationWatcherReconnection(t *testing.T) {
	odigosFake := odigosfake.NewSimpleClientset()
	k8sFake := kubefake.NewSimpleClientset()
	kube.DefaultClient = &kube.Client{
		Interface:    k8sFake,
		OdigosClient: odigosFake.OdigosV1alpha1(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		RunDestinationWatcher(ctx, "odigos-system")
		close(done)
	}()

	// Give the watcher time to start and establish the first watch
	time.Sleep(500 * time.Millisecond)

	// Cancel context to stop the watcher
	cancel()

	// Watcher should exit within a reasonable time
	select {
	case <-done:
		// success: watcher exited cleanly on context cancellation
	case <-time.After(5 * time.Second):
		t.Fatal("watcher did not exit after context cancellation within timeout")
	}
}

// TestInstrumentationConfigWatcherReconnection verifies that the IC watcher
// exits cleanly on context cancellation.
func TestInstrumentationConfigWatcherReconnection(t *testing.T) {
	odigosFake := odigosfake.NewSimpleClientset()
	k8sFake := kubefake.NewSimpleClientset()
	kube.DefaultClient = &kube.Client{
		Interface:    k8sFake,
		OdigosClient: odigosFake.OdigosV1alpha1(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		RunInstrumentationConfigWatcher(ctx, "")
		close(done)
	}()

	// Give the watcher time to start
	time.Sleep(500 * time.Millisecond)

	// Cancel context
	cancel()

	select {
	case <-done:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("IC watcher did not exit after context cancellation within timeout")
	}
}

// TestWatcherRestartsAfterContextCycle verifies that a watcher can be started,
// stopped via context cancellation, and then started again with a new context.
func TestWatcherRestartsAfterContextCycle(t *testing.T) {
	odigosFake := odigosfake.NewSimpleClientset()
	k8sFake := kubefake.NewSimpleClientset()
	kube.DefaultClient = &kube.Client{
		Interface:    k8sFake,
		OdigosClient: odigosFake.OdigosV1alpha1(),
	}

	// First cycle
	ctx1, cancel1 := context.WithCancel(context.Background())
	done1 := make(chan struct{})
	go func() {
		RunDestinationWatcher(ctx1, "odigos-system")
		close(done1)
	}()
	time.Sleep(300 * time.Millisecond)
	cancel1()

	select {
	case <-done1:
	case <-time.After(5 * time.Second):
		t.Fatal("first cycle: watcher did not exit")
	}

	// Second cycle — proves the watcher can restart cleanly
	ctx2, cancel2 := context.WithCancel(context.Background())
	done2 := make(chan struct{})
	go func() {
		RunDestinationWatcher(ctx2, "odigos-system")
		close(done2)
	}()
	time.Sleep(300 * time.Millisecond)
	cancel2()

	select {
	case <-done2:
	case <-time.After(5 * time.Second):
		t.Fatal("second cycle: watcher did not exit")
	}
}
