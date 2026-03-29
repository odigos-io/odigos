package collectorprofiles

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileStore_AddProfileData_ResolvesBufferUnderLock(t *testing.T) {
	s := NewProfileStore(5, 600, 1024*1024, time.Second)
	s.StartViewing("default/Deployment/app")

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.AddProfileData("default/Deployment/app", []byte(`{"ok":true}`))
		}()
	}
	wg.Wait()

	chunks := s.GetProfileData("default/Deployment/app")
	require.NotNil(t, chunks)
	n := 0
	for _, c := range chunks {
		n += len(c)
	}
	assert.Greater(t, n, 0)
}

func TestProfileStore_AddProfileData_InactiveKeyNoop(t *testing.T) {
	s := NewProfileStore(5, 600, 1024*1024, time.Second)
	s.AddProfileData("default/Deployment/ghost", []byte("x"))
	assert.Nil(t, s.GetProfileData("default/Deployment/ghost"))
}
