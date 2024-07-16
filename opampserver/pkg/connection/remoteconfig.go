package connection

import (
	"crypto/sha256"

	"github.com/odigos-io/odigos/opampserver/protobufs"
)

func CalcRemoteConfigHash(remoteConfig *protobufs.AgentConfigMap) []byte {

	hash := sha256.New()
	for _, configEntry := range remoteConfig.ConfigMap {
		hash.Write(configEntry.Body)
	}
	return hash.Sum(nil)
}
