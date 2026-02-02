package process

import (
	"fmt"
	"os"
	"path/filepath"
)

// InjectToProcessTempDir copies the file at sourcePath into the /tmp directory
// of the target process identified by pid.
func InjectToProcessTempDir(pid int, sourcePath string) error {
	// verify sourcePath exists, and it points to a file
	srcInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source path: %s - %w", sourcePath, err)
	}
	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("source path is not a regular file: %s", sourcePath)
	}

	// find the target path inside the process's /proc/<pid>/root/tmp directory
	// and make sure it exists
	procRootPath := ProcFilePath(pid, "root")
	tmpDir := os.TempDir()
	destPath := filepath.Join(procRootPath, tmpDir)

	destInfo, err := os.Stat(destPath)
	if err != nil {
		return fmt.Errorf("failed to stat target tmp dir: %s - %w", destPath, err)
	}
	if !destInfo.IsDir() {
		return fmt.Errorf("target tmp dir is not a directory: %s", destPath)
	}

	// copy the file to the target path
	destFilePath := filepath.Join(destPath, filepath.Base(sourcePath))
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %s - %w", sourcePath, err)
	}
	defer sourceFile.Close()

	targetFile, err := os.OpenFile(destFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open target file: %s - %w", destFilePath, err)
	}
	defer targetFile.Close()

	// ReadFrom use copy_file_range or splice under the hood for zero-copy file transfer
	// see https://man7.org/linux/man-pages/man2/copy_file_range.2.html#ERRORS
	if _, err = targetFile.ReadFrom(sourceFile); err != nil {
		return fmt.Errorf("failed to copy file to target: %s - %w", destFilePath, err)
	}

	return nil
}
