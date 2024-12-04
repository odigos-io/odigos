package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/odigos-io/odigos/opampserver/protobufs"
)

func TestCalcRemoteConfigHashConsistent(t *testing.T) {
	remoteConfig := protobufs.AgentConfigMap{
		ConfigMap: map[string]*protobufs.AgentConfigFile{
			"key1": {
				Body: []byte("value1"),
			},
			"key2": {
				Body: []byte("value2"),
			},
			"key3": {
				Body: []byte("value3"),
			},
			"key4": {
				Body: []byte("value4"),
			},
			"key5": {
				Body: []byte("value5"),
			},
		},
	}

	hash1 := CalcRemoteConfigHash(&remoteConfig)
	hash2 := CalcRemoteConfigHash(&remoteConfig)
	assert.Equal(t, hash1, hash2)
}