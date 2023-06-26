package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

//go:generate yarn --cwd ../webapp build

const (
	assetsDir   = "webapp/out"
	defaultPort = 3000
)

func main() {
	var addressFlag string
	var portFlag int
	flag.StringVar(&addressFlag, "address", "localhost", "Address to listen on")
	flag.IntVar(&portFlag, "port", defaultPort, "Port to listen on")
	flag.Parse()

	// Serve all files in `web/` directory
	http.Handle("/", http.FileServer(http.Dir(assetsDir)))
	http.Handle("/api/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write json response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Hello world!"}`))
	}))

	// Start server
	log.Println("Starting Odigos UI...")
	log.Printf("Odigos UI is available at: http://%s:%d", addressFlag, portFlag)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", addressFlag, portFlag), nil)
	if err != nil {
		log.Printf("Error starting server: %s", err)
		os.Exit(-1)
	}
}
