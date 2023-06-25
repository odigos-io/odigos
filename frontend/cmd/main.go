package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

//go:generate yarn --cwd ../webapp build

const (
	assetsDir = "webapp/out"
	port      = 3000
)

func main() {
	//var addressFlag string
	//var portFlag int
	//flag.StringVar(&addressFlag, "address", "localhost", "Address to listen on")
	//flag.IntVar(&portFlag, "port", port, "Port to listen on")

	// Serve all files in `web/` directory
	http.Handle("/", http.FileServer(http.Dir(assetsDir)))
	http.Handle("/api/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write json response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Hello world!"}`))
	}))

	// Start server
	log.Println("Starting Odigos UI...")
	log.Printf("Odigos UI is available at: http://localhost:%d", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Printf("Error starting server: %s", err)
		os.Exit(-1)
	}
}
