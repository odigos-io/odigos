// Command rust-versions-gen keeps
// procdiscovery/pkg/inspectors/rust/rust_versions.yaml up to date with the
// rustc commit hash -> release version mapping published as tags on
// rust-lang/rust.
//
// This lives outside procdiscovery on purpose: procdiscovery is meant to be
// a package consumed by other parts of the codebase, not a home for
// generator tooling. It is invoked either via `go generate` (see the
// directive in procdiscovery/pkg/inspectors/rust/rust_versions.go) or
// directly from CI (see .github/workflows/update-rust-versions.yml).
//
// Rather than re-encoding the whole mapping on every run, this only appends
// newly discovered hash/version pairs to the end of the file. That keeps
// the diff on each automated PR limited to the handful of new lines added,
// instead of reformatting/reordering entries that didn't change.
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// outputPath is relative to this directory (scripts/rust-versions-gen),
// which is where this program is expected to be run from - either via
// `go run main.go` after `cd`-ing here (matching the convention used by the
// other scripts/* generators), or via `go generate` in
// procdiscovery/pkg/inspectors/rust, which shells out with `cd` first.
const outputPath = "../../procdiscovery/pkg/inspectors/rust/rust_versions.yaml"

func main() {
	log.Println("Fetching rust-lang/rust tags...")
	discovered, err := fetchVersionsFromTags()
	if err != nil {
		log.Fatalf("Failed to fetch rust-lang/rust tags: %v", err)
	}
	log.Printf("Found %d rust versions from upstream tags.", len(discovered))

	existing, err := loadExisting(outputPath)
	if err != nil {
		log.Fatalf("Failed to read existing %s: %v", outputPath, err)
	}

	added, err := appendNewEntries(outputPath, existing, discovered)
	if err != nil {
		log.Fatalf("Failed to update %s: %v", outputPath, err)
	}

	log.Printf("Appended %d new rust version(s) to %s (%d total).", added, outputPath, len(existing)+added)
}

// fetchVersionsFromTags queries rust-lang/rust's tags via `git ls-remote` and
// returns a map of rustc commit hash -> semantic release version.
func fetchVersionsFromTags() (map[string]string, error) {
	cmd := exec.Command("git", "ls-remote", "--tags", "https://github.com/rust-lang/rust.git")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git ls-remote: %w", err)
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
		if match == nil {
			continue
		}

		version := match[1]
		// git ls-remote sorts refs alphabetically, so 'refs/tags/1.0.0'
		// comes before 'refs/tags/1.0.0^{}'. Overwriting versionToHash[version]
		// lets the ^{} commit hash naturally replace the tag object hash.
		versionToHash[version] = hash
	}

	discovered := make(map[string]string, len(versionToHash))
	for version, hash := range versionToHash {
		discovered[hash] = version
	}
	return discovered, nil
}

// loadExisting reads the current hash -> version mapping from path. A
// missing file is treated as an empty mapping (e.g. first run).
func loadExisting(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}

	existing := make(map[string]string)
	if err := yaml.Unmarshal(data, &existing); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return existing, nil
}

// appendNewEntries writes only the hashes from discovered that are not
// already present in existing to the end of the file at path, one
// "hash: \"version\"" line at a time, leaving every existing line untouched.
// It returns the number of entries appended.
func appendNewEntries(path string, existing, discovered map[string]string) (int, error) {
	newHashes := make([]string, 0, len(discovered))
	for hash := range discovered {
		if _, ok := existing[hash]; ok {
			continue
		}
		newHashes = append(newHashes, hash)
	}
	if len(newHashes) == 0 {
		return 0, nil
	}
	// Sort for a deterministic, reviewable diff across runs.
	sort.Strings(newHashes)

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	for _, hash := range newHashes {
		if _, err := fmt.Fprintf(file, "%s: %q\n", hash, discovered[hash]); err != nil {
			return 0, err
		}
	}

	return len(newHashes), nil
}
