// Command symbolize-server runs the node-local native symbolization server.
// vm-agent (VMs) and odiglet (k8s) run this so the collector can symbolize native
// profile frames over a unix socket without doing ELF analysis in its own pipeline.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/odigos-io/odigos/profiles/symbolizeserver"
)

func main() {
	socket := flag.String("socket", symbolizeserver.DefaultSocketPath, "unix socket to listen on")
	debug := flag.Bool("debug", false, "debug logging")
	flag.Parse()

	cfg := zap.NewProductionConfig()
	if *debug {
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}
	log, _ := cfg.Build()
	defer func() { _ = log.Sync() }()

	srv := symbolizeserver.New(*socket, log)
	if err := srv.Start(); err != nil {
		log.Fatal("failed to start symbolize server", zap.Error(err))
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	srv.Close(context.Background())
}
