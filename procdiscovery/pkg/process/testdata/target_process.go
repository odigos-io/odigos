package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <filename> <expected-content>\n", os.Args[0])
		os.Exit(1)
	}

	filename := os.Args[1]
	expectedContent := os.Args[2]

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1)

	fmt.Println("Target process started, waiting for SIGUSR1...")

	select {
	case <-sigChan:
		fmt.Println("Received SIGUSR1, checking file...")
	case <-time.After(10 * time.Second):
		fmt.Fprintf(os.Stderr, "ERROR: timeout waiting for SIGUSR1\n")
		os.Exit(1)
	}

	// Now check for the file
	tmpDir := os.TempDir()
	targetFile := filepath.Join(tmpDir, filename)

	// Try multiple times in case the file isn't immediately visible
	var content []byte
	var err error
	for i := 0; i < 10; i++ {
		content, err = os.ReadFile(targetFile)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to read file %s: %v\n", targetFile, err)
		os.Exit(1)
	}

	if string(content) != expectedContent {
		fmt.Fprintf(os.Stderr, "ERROR: content mismatch: got=%q, want=%q\n", content, expectedContent)
		os.Exit(1)
	}

	fmt.Println("SUCCESS: file verified in target process")
}