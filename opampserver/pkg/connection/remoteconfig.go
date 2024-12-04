package connection

import (
	"crypto/sha256"
	"slices"

	"github.com/odigos-io/odigos/opampserver/protobufs"
)

func CalcRemoteConfigHash(remoteConfig *protobufs.AgentConfigMap) []byte {
	// sort the entries by key
	// then hash the body of each entry
	// this is required to ensure that the hash is consistent
	sortedKeys := make([]string, 0, len(remoteConfig.ConfigMap))
	for key := range remoteConfig.ConfigMap {
		sortedKeys = append(sortedKeys, key)
	}

	slices.Sort(sortedKeys)

	hash := sha256.New()
	for _, key := range sortedKeys {
		configEntry := remoteConfig.ConfigMap[key]
		hash.Write(configEntry.Body)
	}

	return hash.Sum(nil)
}
