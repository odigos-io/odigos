package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/odigos-io/odigos/opampserver/pkg/opampserver"
)

func main() {
	fmt.Println("Starting opampserver...")

	err := opampserver.StartOpAmpServer()
	if err != nil {
		fmt.Println(err)
	}

	// Create a channel to listen for an interrupt or termination signal from the OS.
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Block until a signal is received.
	<-stopChan

	// opampsrv.Stop(context.Background())
}
