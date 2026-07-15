package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	log.Println("Fetching rust-lang/rust tags...")
	cmd := exec.Command("git", "ls-remote", "--tags", "https://github.com/rust-lang/rust.git")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to run git ls-remote: %v", err)
	}

	versionToHash := make(map[string]string)
	semverRe := regexp.MustCompile(`^refs/tags/([0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?)$`)

	scanner := bytes.NewBuffer(out.Bytes())
	for {
		line, err := scanner.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		hash := parts[0]
		ref := parts[1]

		isAnnotated := strings.HasSuffix(ref, "^{}")
		if isAnnotated {
			ref = strings.TrimSuffix(ref, "^{}")
		}
		match := semverRe.FindStringSubmatch(ref)
		if match != nil {
			version := match[1]
			// Since git ls-remote sorts refs alphabetically, 'refs/tags/1.0.0' comes before 'refs/tags/1.0.0^{}'.
			// By overwriting versionToHash[version], the ^{} commit hash naturally replaces the tag object hash!
			versionToHash[version] = hash
		}
	}

	finalMap := make(map[string]string)
	for v, h := range versionToHash {
		finalMap[h] = v
	}

	log.Printf("Found %d rust versions.", len(finalMap))

	// The script will be run from `procdiscovery/pkg/inspectors/rust/generate/`
	// so the JSON should go to `procdiscovery/pkg/inspectors/rust/rust_versions.json`
	jsonPath := filepath.Join("..", "rust_versions.json")
	
	file, err := os.Create(jsonPath)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(finalMap); err != nil {
		log.Fatalf("Failed to encode JSON: %v", err)
	}

	log.Printf("Successfully wrote %s", jsonPath)
}
