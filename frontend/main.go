package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"

	"github.com/keyval-dev/odigos/frontend/kube"

	"k8s.io/client-go/util/homedir"

	"github.com/keyval-dev/odigos/frontend/endpoints"

	"github.com/gin-gonic/gin"
)

const (
	defaultPort = 3000
)

type Flags struct {
	Address    string
	Port       int
	Debug      bool
	KubeConfig string
}

//go:embed all:webapp/out/*
var uiFS embed.FS

func parseFlags() Flags {
	defaultKubeConfig := ""
	if home := homedir.HomeDir(); home != "" {
		defaultKubeConfig = filepath.Join(home, ".kube", "config")
	}

	var flags Flags
	flag.StringVar(&flags.Address, "address", "localhost", "Address to listen on")
	flag.IntVar(&flags.Port, "port", defaultPort, "Port to listen on")
	flag.BoolVar(&flags.Debug, "debug", false, "Enable debug mode")
	flag.StringVar(&flags.KubeConfig, "kubeconfig", defaultKubeConfig, "Path to kubeconfig file")
	flag.Parse()
	return flags
}

func initKubernetesClient(flags *Flags) error {
	client, err := kube.CreateClient(flags.KubeConfig)
	if err != nil {
		return fmt.Errorf("error creating Kubernetes client: %w", err)
	}

	kube.SetDefaultClient(client)
	return nil
}

func startHTTPServer(flags *Flags) (*gin.Engine, error) {
	var r *gin.Engine
	if flags.Debug {
		r = gin.Default()
	} else {
		gin.SetMode(gin.ReleaseMode)
		r = gin.New()
		r.Use(gin.Recovery())
	}

	// Serve React app
	dist, err := fs.Sub(uiFS, "webapp/out")
	if err != nil {
		return nil, fmt.Errorf("error reading webapp/out directory: %s", err)
	}
	r.NoRoute(gin.WrapH(http.FileServer(http.FS(dist))))

	// Serve API
	apis := r.Group("/api")
	{
		apis.GET("/namespaces", endpoints.GetNamespaces)
		apis.POST("/namespaces", endpoints.PersistNamespaces)
		apis.GET("/applications/:namespace", endpoints.GetApplicationsInNamespace)
	}

	return r, nil
}

func main() {
	flags := parseFlags()

	// Connect to Kubernetes
	err := initKubernetesClient(&flags)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %s", err)
	}

	// Start server
	r, err := startHTTPServer(&flags)
	if err != nil {
		log.Fatalf("Error starting server: %s", err)
	}

	log.Println("Starting Odigos UI...")
	log.Printf("Odigos UI is available at: http://%s:%d", flags.Address, flags.Port)
	err = r.Run(fmt.Sprintf("%s:%d", flags.Address, flags.Port))
	if err != nil {
		log.Fatalf("Error starting server: %s", err)
	}
}
