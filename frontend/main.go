package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
)

const (
	defaultPort = 3000
)

//go:embed all:webapp/out/*
var uiFS embed.FS

func main() {
	var addressFlag string
	var portFlag int
	flag.StringVar(&addressFlag, "address", "localhost", "Address to listen on")
	flag.IntVar(&portFlag, "port", defaultPort, "Port to listen on")
	flag.Parse()

	// Serve all files in `web/` directory
	dist, err := fs.Sub(uiFS, "webapp/out")
	if err != nil {
		log.Fatalf("Error reading webapp/out directory: %s", err)
	}

	http.Handle("/", http.FileServer(http.FS(dist)))
	http.Handle("/api/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write json response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Hello world!"}`))
	}))

	// Start server
	log.Println("Starting Odigos UI...")
	log.Printf("Odigos UI is available at: http://%s:%d", addressFlag, portFlag)
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", addressFlag, portFlag), nil)
	if err != nil {
		log.Printf("Error starting server: %s", err)
		os.Exit(-1)
	}
}
