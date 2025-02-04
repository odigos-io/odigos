package workloadrollout

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"slices"
	"strings"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

func hashForContainersConfig(containersConfig []odigosv1alpha1.ContainerConfig) ([]byte, error) {
	if len(containersConfig) == 0 {
		return []byte{}, nil
	}

	// sort the entries by container name
	// then hash the body of each entry
	// this is required to ensure that the hash is consistent
	slices.SortFunc(containersConfig, func(i, j odigosv1alpha1.ContainerConfig) int {
		return strings.Compare(i.ContainerName, j.ContainerName)
	})

	hash := sha256.New()
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	var err error

	// take only the relevant fields for computing the hash
	type configHashFields struct {
		ContainerName    string
		Instrumented     bool
		OtelDistroName   string
	}

	for i := range containersConfig {
		configEntry := configHashFields{
			ContainerName:  containersConfig[i].ContainerName,
			Instrumented:   containersConfig[i].Instrumented,
			OtelDistroName: containersConfig[i].OtelDistroName,
		}
		err = enc.Encode(configEntry)
		if err != nil {
			return nil, err
		}
		hash.Write(buf.Bytes())
		buf.Reset()
	}

	return hash.Sum(nil), nil
}