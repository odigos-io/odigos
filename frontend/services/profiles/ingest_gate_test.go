package profiles

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProfilesIngestGateStartsInTheRequestedState(t *testing.T) {
	assert.True(t, NewProfilesIngestGate(true).IsEnabled())
	assert.False(t, NewProfilesIngestGate(false).IsEnabled())
}

func TestIngestGateSetFlipsTheState(t *testing.T) {
	gate := NewProfilesIngestGate(false)

	gate.Set(true)
	assert.True(t, gate.IsEnabled())

	gate.Set(false)
	assert.False(t, gate.IsEnabled())
}

// The gate is flipped by the effective-config watcher while the OTLP receiver is concurrently
// consuming batches, so reads and writes must not race.
func TestIngestGateIsSafeForConcurrentUse(t *testing.T) {
	gate := NewProfilesIngestGate(false)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func(enable bool) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				gate.Set(enable)
			}
		}(i%2 == 0)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = gate.IsEnabled()
			}
		}()
	}
	wg.Wait()

	gate.Set(true)
	assert.True(t, gate.IsEnabled())
}
