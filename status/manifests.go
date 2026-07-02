package status

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func unmarshalManifest(data []byte) (Status, error) {
	var s Status
	if err := yaml.Unmarshal(data, &s); err != nil {
		return Status{}, err
	}
	return s, nil
}

func isManifestFile(name string) bool {
	return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
}

// LoadManifestsFromDir loads all status YAML manifests under dir.
func LoadManifestsFromDir(dir string) ([]Status, error) {
	var manifests []Status
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !isManifestFile(path) {
			return nil
		}

		bytesData, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		s, err := unmarshalManifest(bytesData)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		manifests = append(manifests, s)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return manifests, nil
}

// LoadManifestsFromFS loads all status YAML manifests under dir within fsys.
func LoadManifestsFromFS(fsys fs.FS, dir string) ([]Status, error) {
	var manifests []Status
	err := fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !isManifestFile(path) {
			return nil
		}

		bytesData, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		s, err := unmarshalManifest(bytesData)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		manifests = append(manifests, s)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return manifests, nil
}
