package main

import (
	"fmt"
	"log"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra/doc"

	"github.com/odigos-io/odigos/cli/cmd"
)

const fmTemplate = `---
title: "%s"
sidebarTitle: "%s"
---
`

func main() {
	filePrepender := func(filename string) string {
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		command := strings.Replace(base, "_", " ", -1)
		return fmt.Sprintf(fmTemplate, command, command)
	}

	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		return "/cli/" + strings.ToLower(base)
	}

	rootCmd := cmd.RootCmd()
	err := doc.GenMarkdownTreeCustom(&rootCmd, "../../docs/cli", filePrepender, linkHandler)
	if err != nil {
		log.Fatal(err)
	}
}
