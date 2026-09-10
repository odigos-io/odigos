// Command odigos-browser-proxy runs as a sidecar in front of a browser-instrumented web server.
//
// It has two modes:
//
//	odigos-browser-proxy        - run the proxy server (default; the sidecar container)
//	odigos-browser-proxy init   - apply the iptables inbound redirect (the init container)
package main

import (
	"log"
	"os"

	"github.com/odigos-io/odigos/browser-proxy/internal/config"
	"github.com/odigos-io/odigos/browser-proxy/internal/iptables"
	"github.com/odigos-io/odigos/browser-proxy/internal/server"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		runInit()
		return
	}
	runServe()
}

func runInit() {
	cfg, err := config.LoadInit()
	if err != nil {
		log.Fatalf("browser-proxy init: invalid configuration: %v", err)
	}
	if err := iptables.Apply(iptables.Config{
		AppPort:   cfg.AppPort,
		ProxyPort: cfg.ProxyPort,
		ProxyUID:  cfg.ProxyUID,
	}); err != nil {
		log.Fatalf("browser-proxy init: failed to apply iptables redirect: %v", err)
	}
	log.Printf("browser-proxy init: redirected inbound tcp/%d -> sidecar tcp/%d", cfg.AppPort, cfg.ProxyPort)
}

func runServe() {
	cfg, err := config.LoadServe()
	if err != nil {
		log.Fatalf("browser-proxy: invalid configuration: %v", err)
	}
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("browser-proxy: failed to start: %v", err)
	}
	if err := srv.Run(); err != nil {
		log.Fatalf("browser-proxy: server exited: %v", err)
	}
}
