package agentenabled

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"slices"
	"strings"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

func hashForContainersConfig(containersConfig []odigosv1.ContainerAgentConfig) (string, error) {
	if len(containersConfig) == 0 {
		return "", nil
	}

	// sort the entries by container name
	// then hash the body of each entry
	// this is required to ensure that the hash is consistent
	slices.SortFunc(containersConfig, func(i, j odigosv1.ContainerAgentConfig) int {
		return strings.Compare(i.ContainerName, j.ContainerName)
	})

	hash := sha256.New()
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	var err error

	// take only the relevant fields for computing the hash
	type configHashFields struct {
		ContainerName  string
		Instrumented   bool
		OtelDistroName string
	}

	for i := range containersConfig {
		configEntry := configHashFields{
			ContainerName:  containersConfig[i].ContainerName,
			Instrumented:   containersConfig[i].AgentEnabled,
			OtelDistroName: containersConfig[i].OtelDistroName,
		}
		err = enc.Encode(configEntry)
		if err != nil {
			return "", err
		}
		hash.Write(buf.Bytes())
		buf.Reset()
	}

	hashBytes := hash.Sum(nil)
	// limit the hash to 16 characters, as it's written into k8s resources and annotations
	shortHash := hashBytes[:8]
	hashAsHex := hex.EncodeToString(shortHash)
	return hashAsHex, nil
}
