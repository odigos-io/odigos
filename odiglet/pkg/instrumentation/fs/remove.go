package fs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/odigos-io/odigos/odiglet/pkg/log"
)

func removeFilesInDir(hostDir string) error {
	keepCFiles := !ShouldRecreateAllCFiles()
	log.Logger.V(0).Info(fmt.Sprintf("Removing files in directory: %s, keepCFiles: %s", hostDir, fmt.Sprintf("%t", keepCFiles)))

	return filepath.Walk(hostDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself]
		if path == hostDir {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if keepCFiles {
			switch ext := filepath.Ext(info.Name()); ext {
			case ".so", ".node", "node.d", ".a":
				return nil
			}
		}

		// Remove the file
		err = os.Remove(path)
		if err != nil {
			return err
		}

		return nil
	})
}
