package fs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/odigos-io/odigos/odiglet/pkg/log"
)

func removeFilesInDir(hostDir string, unchangedFiles map[string]struct{}, filesToKeep map[string]struct{}) error {
	log.Logger.V(0).Info("Removing files in the host directory", "hostDir", hostDir)

	// Protect directories that contain files we want to keep and unchanged files
	protectedDirs := calculateProtectedDirs(hostDir, filesToKeep, unchangedFiles)

	return filepath.WalkDir(hostDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == hostDir {
			return nil
		}

		// Skip removing files that are either in the filesToKeep map, have a versioning suffix, or not changed
		if !d.IsDir() {
			if _, keep := filesToKeep[path]; keep {
				log.Logger.V(0).Info("Skipping protected file", "file", path)
				return nil
			}

			if strings.Contains(path, "_hash_version-") {
				log.Logger.V(0).Info("Skipping file with versioning suffix", "file", path)
				return nil
			}
			if _, isUnchanged := unchangedFiles[path]; isUnchanged {
				log.Logger.V(1).Info("Skipping unchanged file", "file", path)
				return nil
			}
		}

		if d.IsDir() {
			if _, protected := protectedDirs[path]; protected {
				log.Logger.V(1).Info("Skipping protected directory", "directory", path)
				return nil
			}
		}

		err = os.RemoveAll(path)
		if err != nil {
			return fmt.Errorf("error removing %s: %w", path, err)
		}

		// Skip further processing in this directory since it has been removed
		if d.IsDir() {
			return filepath.SkipDir
		}

		return nil
	})
}

func calculateProtectedDirs(hostDir string, filesToKeep, unchangedFiles map[string]struct{}) map[string]struct{} {
	protectedDirs := make(map[string]struct{})

	// Always protect the root hostDir itself
	protectedDirs[hostDir] = struct{}{}

	addDirs := func(file string) {
		dir := filepath.Dir(file)
		for dir != hostDir {
			protectedDirs[dir] = struct{}{}
			dir = filepath.Dir(dir)
		}
	}

	for file := range filesToKeep {
		addDirs(file)
	}

	for file := range unchangedFiles {
		addDirs(file)
	}

	return protectedDirs
}
