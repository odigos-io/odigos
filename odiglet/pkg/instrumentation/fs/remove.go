package fs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/odigos-io/odigos/odiglet/pkg/log"
)

func removeFilesInDir(hostDir string, filesToKeep map[string]struct{}) error {
	log.Logger.V(0).Info("Removing files in the host directory", "hostDir", hostDir)

	// Mark directories as protected if they contain a file that needs to be preserved.
	protectedDirs := make(map[string]bool)
	for file := range filesToKeep {
		dir := filepath.Dir(file)
		for dir != hostDir {
			protectedDirs[dir] = true
			dir = filepath.Dir(dir)
		}
		protectedDirs[hostDir] = true // Protect the main directory
	}

	return filepath.Walk(hostDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == hostDir {
			return nil
		}

		// Skip removing any files listed in filesToKeepMap
		if !info.IsDir() {
			if _, found := filesToKeep[path]; found {
				log.Logger.V(0).Info("Skipping protected file", "file", path)
				return nil
			}
		}

		// Skip removing protected directories
		if info.IsDir() && protectedDirs[path] {
			log.Logger.V(0).Info("Skipping protected directory", "directory", path)
			return nil
		}

		// Remove removing unprotected files and directories
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("error removing %s: %w", path, err)
		}

		// Skip further processing in this directory since it has been removed
		if info.IsDir() {
			return filepath.SkipDir
		}

		return nil
	})
}
