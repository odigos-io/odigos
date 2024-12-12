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
	"sync"

	"github.com/fsnotify/fsnotify"
	"go.opentelemetry.io/collector/confmap"
	"go.uber.org/zap"
)

const schemeName = "file"

type provider struct {
	wg sync.WaitGroup
	mu sync.Mutex

	watcher *fsnotify.Watcher
	running bool
	logger  *zap.Logger
}

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
//
// This provider is forked from the default upstream OSS fileprovider (go.opentelemetry.io/collector/confmap/provider/fileprovider)
// to provide file watching and reloading. It is exactly the same except it uses fsnotify to watch
// for changes to the config file in an infinite routine. When a change is found, the confmap.WatcherFunc
// is called to signal the collector to reload its config.
// Because Odigos mounts collecotr configs from a ConfigMap, the mounted file is a symlink. So we watch for
// add/remove events (rather than write events). Kubernetes automatically updates the projected contents when
// the configmap changes. This lets us use new config changes without restarting the collector deployment.
func NewFactory() confmap.ProviderFactory {
	return confmap.NewProviderFactory(newProvider)
}

func newProvider(c confmap.ProviderSettings) confmap.Provider {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		c.Logger.Error("unable to start fsnotify watcher", zap.Error(err))
	}
	return &provider{
		logger:  c.Logger,
		watcher: watcher,
		running: false,
	}
}

func (fmp *provider) Retrieve(ctx context.Context, uri string, wf confmap.WatcherFunc) (*confmap.Retrieved, error) {
	fmp.mu.Lock()
	defer fmp.mu.Unlock()

	if !strings.HasPrefix(uri, schemeName+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}

	// Clean the path before using it.
	file := filepath.Clean(uri[len(schemeName)+1:])
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read the file %v: %w", uri, err)
	}

	err = fmp.watcher.Add(file)
	if err != nil {
		return nil, err
	}

	// start a new watcher routine only if one isn't already running, since Retrieve could be called multiple times
	if !fmp.running {
		fmp.running = true
		fmp.wg.Add(1)
		go func() {
			defer fmp.wg.Done()
		LOOP:
			for {
				select {
				case event, ok := <-fmp.watcher.Events:
					if !ok {
						fmp.logger.Info("watch channel closed")
						break LOOP
					}
					// k8s configmaps are mounted as symlinks; need to watch for remove, not write
					if event.Has(fsnotify.Remove) {
						fmp.watcher.Remove(file)
						fmp.watcher.Add(file)
						wf(&confmap.ChangeEvent{})
					}

				case err, ok := <-fmp.watcher.Errors:
					if !ok {
						fmp.logger.Info("fsnotify error channel closed")
						break LOOP
					}
					wf(&confmap.ChangeEvent{Error: fmt.Errorf("error watching event %+v", err)})

				case <-ctx.Done():
					err := fmp.watcher.Close()
					if err != nil {
						fmp.logger.Error("error closing fsnotify watcher", zap.Error(err))
					}
					break LOOP
				}
			}
			fmp.mu.Lock()
			fmp.running = false
			fmp.mu.Unlock()
		}()
	}

	return confmap.NewRetrievedFromYAML(content)
}

func (*provider) Scheme() string {
	return schemeName
}

func (fmp *provider) Shutdown(context.Context) error {
	// close watcher channels
	err := fmp.watcher.Close()
	if err != nil {
		fmp.logger.Error("error closing fsnotify watcher", zap.Error(err))
	}
	// wait for watcher routine to finish
	fmp.wg.Wait()
	return nil
}
