// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// forked from go.opentelemetry.io/collector/confmap/provider/fileprovider
package odigosfileprovider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"go.opentelemetry.io/collector/confmap"
)

const schemeName = "file"

type provider struct{}

// NewFactory returns a factory for a confmap.Provider that reads the configuration from a file.
//
// This Provider supports "file" scheme, and can be called with a "uri" that follows:
//
//	file-uri		= "file:" local-path
//	local-path		= [ drive-letter ] file-path
//	drive-letter	= ALPHA ":"
//
// The "file-path" can be relative or absolute, and it can be any OS supported format.
//
// Examples:
// `file:path/to/file` - relative path (unix, windows)
// `file:/path/to/file` - absolute path (unix, windows)
// `file:c:/path/to/file` - absolute path including drive-letter (windows)
// `file:c:\path\to\file` - absolute path including drive-letter (windows)
func NewFactory() confmap.ProviderFactory {
	return confmap.NewProviderFactory(newProvider)
}

func newProvider(confmap.ProviderSettings) confmap.Provider {
	return &provider{}
}

func (fmp *provider) Retrieve(_ context.Context, uri string, wf confmap.WatcherFunc) (*confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}

	// Clean the path before using it.
	file := filepath.Clean(uri[len(schemeName)+1:])
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read the file %v: %w", uri, err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = watcher.Add(file)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					wf(&confmap.ChangeEvent{Error: fmt.Errorf("error watching event")})
				}
				if event.Has(fsnotify.Write) {
					fmt.Printf("modified file: %+v", event.Name)
					wf(&confmap.ChangeEvent{})
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					wf(&confmap.ChangeEvent{Error: fmt.Errorf("error watching event %+v", err)})
				}
			}
		}
	}()

	return confmap.NewRetrievedFromYAML(content)
}

func (*provider) Scheme() string {
	return schemeName
}

func (*provider) Shutdown(context.Context) error {
	return nil
}
