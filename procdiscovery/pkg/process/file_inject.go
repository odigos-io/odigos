package process

import (
	"fmt"
	"os"
	"path/filepath"
)

func procTempPath(pid int) (string, error) {
	// find the target path inside the process's /proc/<pid>/root/tmp directory
	// and make sure it exists
	procRootPath := ProcFilePath(pid, "root")
	tmpDir := os.TempDir()
	destPath := filepath.Join(procRootPath, tmpDir)

	destInfo, err := os.Stat(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat target tmp dir: %s - %w", destPath, err)
	}
	if !destInfo.IsDir() {
		return "", fmt.Errorf("target tmp dir is not a directory: %s", destPath)
	}
	return destPath, nil
}

// InjectFileToProcessTempDir copies the file at sourcePath into the /tmp directory
// of the target process identified by pid.
func InjectFileToProcessTempDir(pid int, sourcePath string) error {
	srcInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source path: %s - %w", sourcePath, err)
	}
	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("source path is not a regular file: %s", sourcePath)
	}

	destTempPath, err := procTempPath(pid)
	if err != nil {
		return err
	}

	// copy the file to the target path
	destFilePath := filepath.Join(destTempPath, filepath.Base(sourcePath))
	return copyFile(sourcePath, destFilePath, srcInfo.Mode().Perm())
}

// InjectDirToProcessTempDir copies a complete directory to the process's temp directory.
// It preserves directory structure and file permissions.
func InjectDirToProcessTempDir(pid int, sourceDirPath string) error {
	info, err := os.Stat(sourceDirPath)
	if err != nil {
		return fmt.Errorf("failed to stat source directory: %s - %w", sourceDirPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", sourceDirPath)
	}

	destTempPath, err := procTempPath(pid)
	if err != nil {
		return err
	}

	// Create the destination directory with the same name as source
	destDirPath := filepath.Join(destTempPath, filepath.Base(sourceDirPath))
	if err := os.MkdirAll(destDirPath, info.Mode().Perm()); err != nil {
		return fmt.Errorf("failed to create target directory: %s - %w", destDirPath, err)
	}

	// Walk through the source directory and copy regular files and directories
	return filepath.Walk(sourceDirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk error at %s: %w", path, err)
		}

		relPath, err := filepath.Rel(sourceDirPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		targetPath := filepath.Join(destDirPath, relPath)

		switch {
		case relPath == ".":
			return nil
		case info.IsDir():
			if err := os.MkdirAll(targetPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			return nil
		case info.Mode().IsRegular():
			return copyFile(path, targetPath, info.Mode().Perm())
		default:
			return nil
		}
	})
}

// copyFile copies a file from sourcePath to destPath with the given permissions.
func copyFile(sourcePath, destPath string, perm os.FileMode) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %s - %w", sourcePath, err)
	}
	defer sourceFile.Close()

	targetFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to open target file: %s - %w", destPath, err)
	}
	defer targetFile.Close()

	// ReadFrom use copy_file_range or splice under the hood for zero-copy file transfer
	// see https://man7.org/linux/man-pages/man2/copy_file_range.2.html#ERRORS
	if _, err = targetFile.ReadFrom(sourceFile); err != nil {
		return fmt.Errorf("failed to copy file to target: %s - %w", destPath, err)
	}

	return nil
}
