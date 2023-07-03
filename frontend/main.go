package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const (
	defaultPort = 3000
)

//go:embed all:webapp/out/*
var uiFS embed.FS

func main() {
	var addressFlag string
	var portFlag int
	var debugFlag bool
	var kubeConfig string
	flag.StringVar(&addressFlag, "address", "localhost", "Address to listen on")
	flag.IntVar(&portFlag, "port", defaultPort, "Port to listen on")
	flag.BoolVar(&debugFlag, "debug", false, "Enable debug mode")
	flag.StringVar(&kubeConfig, "kubeconfig", "", "Path to kubeconfig file")
	flag.Parse()

	// Serve all files in `web/` directory
	dist, err := fs.Sub(uiFS, "webapp/out")
	if err != nil {
		log.Fatalf("Error reading webapp/out directory: %s", err)
	}

	var r *gin.Engine
	if debugFlag {
		r = gin.Default()
	} else {
		gin.SetMode(gin.ReleaseMode)
		r = gin.New()
		r.Use(gin.Recovery())
	}

	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Hello world!",
		})
	})
	r.NoRoute(gin.WrapH(http.FileServer(http.FS(dist))))

	// Start server
	log.Println("Starting Odigos UI...")
	log.Printf("Odigos UI is available at: http://%s:%d", addressFlag, portFlag)
	err = r.Run(fmt.Sprintf("%s:%d", addressFlag, portFlag))
	if err != nil {
		log.Printf("Error starting server: %s", err)
		os.Exit(-1)
	}
}
