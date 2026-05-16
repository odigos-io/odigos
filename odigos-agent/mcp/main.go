package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/odigos-io/odigos-agent/mcp/server"
)

func main() {
	listenAddr := os.Getenv("MCP_CLUSTER_LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = "0.0.0.0:9090"
	}

	httpServer, err := server.New()
	if err != nil {
		log.Fatalf("build server: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("odigos-agent cluster MCP listening on %s (endpoint /mcp)", listenAddr)
		errCh <- httpServer.Start(listenAddr)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil {
			log.Fatalf("server error: %v", err)
		}
	case sig := <-sigCh:
		log.Printf("shutdown requested (%s)", sig)
		// Bound shutdown so a stuck streamable-HTTP request can't hold us past
		// the kubelet's terminationGracePeriodSeconds.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Fatalf("shutdown error: %v", err)
		}
		fmt.Println("bye")
	}
}
