package main

import (
	"bytes"
	"debug/elf"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: detect-local <binary-path>")
		fmt.Println("Example: detect-local /tmp/rust-stripped")
		os.Exit(1)
	}

	binaryPath := os.Args[1]

	fmt.Printf("╔══════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║           🦀 Rust Binary Detection Test                         ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════════╝\n\n")
	fmt.Printf("Binary: %s\n\n", binaryPath)

	f, err := os.Open(binaryPath)
	if err != nil {
		fmt.Printf("❌ Failed to open binary: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	info, _ := f.Stat()
	fmt.Printf("Size: %d bytes (%.2f MB)\n\n", info.Size(), float64(info.Size())/1024/1024)

	elfFile, err := elf.NewFile(f)
	if err != nil {
		fmt.Printf("⚠️  Not an ELF binary: %v\n", err)
		fmt.Println("Trying panic string detection only...\n")
	}

	detected := false
	detectionMethod := ""

	if elfFile != nil {
		fmt.Println("=== Method 1: Symbol Detection ===")
		if checkSymbols(elfFile) {
			fmt.Println("✅ Rust symbols FOUND")
			detected = true
			detectionMethod = "symbols"
		} else {
			fmt.Println("❌ No Rust symbols (expected for stripped binaries)")
		}
		fmt.Println()

		fmt.Println("=== Method 2: ELF Section Detection ===")
		if checkELFSections(elfFile) {
			fmt.Println("✅ Rust ELF sections FOUND (.rustc or .note.rustc)")
			if !detected {
				detected = true
				detectionMethod = "ELF sections"
			}
		} else {
			fmt.Println("❌ No Rust-specific ELF sections")
		}
		fmt.Println()

		fmt.Println("=== Method 4: Dynamic Library Detection ===")
		if checkRustLibraries(elfFile) {
			fmt.Println("✅ Rust standard library FOUND (libstd-*.so)")
			if !detected {
				detected = true
				detectionMethod = "dynamic libraries"
			}
		} else {
			fmt.Println("❌ No Rust dynamic libraries (static build)")
		}
		fmt.Println()
	}

	fmt.Println("=== Method 3: Panic String Detection ===")
	f.Seek(0, io.SeekStart)
	foundStrings := checkPanicStrings(f)
	if len(foundStrings) > 0 {
		fmt.Println("✅ Rust panic strings FOUND:")
		for _, s := range foundStrings {
			fmt.Printf("   - %s\n", truncate(s, 60))
		}
		if !detected {
			detected = true
			detectionMethod = "panic strings"
		}
	} else {
		fmt.Println("❌ No Rust panic strings found")
	}
	fmt.Println()

	fmt.Println("=== Version Detection ===")
	f.Seek(0, io.SeekStart)
	version := extractVersion(f)
	if version != "" {
		fmt.Printf("✅ Rustc commit: %s\n", version)
	} else {
		fmt.Println("❌ Could not extract rustc version")
	}
	fmt.Println()

	fmt.Println("═══════════════════════════════════════════════════════════════════")
	if detected {
		fmt.Printf("🎉 RESULT: Binary detected as RUST (via %s)\n", detectionMethod)
	} else {
		fmt.Println("⚠️  RESULT: Binary NOT detected as Rust")
	}
	fmt.Println("═══════════════════════════════════════════════════════════════════")
}

func checkSymbols(file *elf.File) bool {
	rustPrefixes := []string{"__rust_", "rust_begin_unwind", "rust_eh_personality"}
	rustMangledPrefixes := []string{"_ZN4core", "_ZN5alloc", "_ZN3std"}

	staticSyms, err := file.Symbols()
	if err == nil {
		for _, sym := range staticSyms {
			for _, prefix := range rustPrefixes {
				if strings.Contains(sym.Name, prefix) {
					return true
				}
			}
			for _, prefix := range rustMangledPrefixes {
				if strings.HasPrefix(sym.Name, prefix) {
					return true
				}
			}
		}
	}

	dynSyms, err := file.DynamicSymbols()
	if err == nil {
		for _, sym := range dynSyms {
			for _, prefix := range rustPrefixes {
				if strings.Contains(sym.Name, prefix) {
					return true
				}
			}
			for _, prefix := range rustMangledPrefixes {
				if strings.HasPrefix(sym.Name, prefix) {
					return true
				}
			}
		}
	}

	return false
}

func checkELFSections(file *elf.File) bool {
	for _, section := range file.Sections {
		if section.Name == ".rustc" || section.Name == ".note.rustc" {
			return true
		}
	}
	return false
}

func checkRustLibraries(file *elf.File) bool {
	libs, err := file.ImportedLibraries()
	if err != nil {
		return false
	}

	for _, lib := range libs {
		if strings.Contains(lib, "libstd-") && strings.Contains(lib, ".so") {
			return true
		}
	}
	return false
}

func checkPanicStrings(f *os.File) []string {
	rustPanicStrings := []string{
		"panicked at",
		"called `Option::unwrap()` on a `None` value",
		"called `Result::unwrap()` on an `Err` value",
		"/rustc/",
		".cargo/registry",
		"rust_begin_unwind",
	}

	content, err := io.ReadAll(f)
	if err != nil {
		return nil
	}

	var found []string
	for _, pattern := range rustPanicStrings {
		if bytes.Contains(content, []byte(pattern)) {
			found = append(found, pattern)
		}
	}
	return found
}

func extractVersion(f *os.File) string {
	content, err := io.ReadAll(f)
	if err != nil {
		return ""
	}

	rustcVersionPrefix := []byte("/rustc/")
	if idx := bytes.Index(content, rustcVersionPrefix); idx != -1 {
		end := idx + len(rustcVersionPrefix) + 40
		if end > len(content) {
			end = len(content)
		}
		versionBytes := content[idx+len(rustcVersionPrefix) : end]
		if nullIdx := bytes.IndexByte(versionBytes, 0); nullIdx != -1 {
			versionBytes = versionBytes[:nullIdx]
		}
		if slashIdx := bytes.IndexByte(versionBytes, '/'); slashIdx != -1 {
			versionBytes = versionBytes[:slashIdx]
		}
		version := string(versionBytes)
		if len(version) > 0 && len(version) <= 40 {
			return version
		}
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

