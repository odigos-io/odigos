package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	_ "net/http/pprof"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/destinations"
	"github.com/odigos-io/odigos/frontend/graph"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/kube/watchers"
	"github.com/odigos-io/odigos/frontend/services"
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
	"github.com/odigos-io/odigos/frontend/services/sse"
	"github.com/odigos-io/odigos/frontend/version"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

const (
	defaultPort = 3000
)

type Flags struct {
	Version     bool
	Address     string
	Port        int
	Debug       bool
	KubeConfig  string
	KubeContext string
	Namespace   string
}

//go:embed all:webapp/out/*
var uiFS embed.FS

func parseFlags() Flags {
	defaultKubeConfig := env.GetDefaultKubeConfigPath()

	var flags Flags
	flag.BoolVar(&flags.Version, "version", false, "Print Odigos UI version.")
	flag.StringVar(&flags.Address, "address", "localhost", "Address to listen on")
	flag.IntVar(&flags.Port, "port", defaultPort, "Port to listen on")
	flag.BoolVar(&flags.Debug, "debug", false, "Enable debug mode")
	flag.StringVar(&flags.KubeConfig, "kubeconfig", defaultKubeConfig, "Path to kubeconfig file")
	flag.StringVar(&flags.KubeContext, "kube-context", "", "Name of the kubeconfig context to use")
	flag.StringVar(&flags.Namespace, "namespace", consts.DefaultOdigosNamespace, "Kubernetes namespace where Odigos is installed")
	flag.Parse()
	return flags
}

func initKubernetesClient(flags *Flags) error {
	client, err := kube.CreateClient(flags.KubeConfig, flags.KubeContext)
	if err != nil {
		return fmt.Errorf("error creating Kubernetes client: %w", err)
	}

	kube.SetDefaultClient(client)
	return nil
}

func startHTTPServer(flags *Flags, odigosMetrics *collectormetrics.OdigosMetricsConsumer) (*gin.Engine, error) {
	var r *gin.Engine
	if flags.Debug {
		r = gin.Default()
	} else {
		gin.SetMode(gin.ReleaseMode)
		r = gin.New()
		r.Use(gin.Recovery())
	}

	// Enable CORS
	r.Use(cors.Default())

	// Serve React app
	dist, err := fs.Sub(uiFS, "webapp/out")
	if err != nil {
		return nil, fmt.Errorf("error reading webapp/out directory: %s", err)
	}

	// Serve React app if page not found serve index.html
	r.NoRoute(gin.WrapH(httpFileServerWith404(http.FS(dist))))

	// GraphQL handlers
	gqlHandler := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			MetricsConsumer: odigosMetrics,
		},
	}))
	r.POST("/graphql", func(c *gin.Context) {
		gqlHandler.ServeHTTP(c.Writer, c.Request)
	})
	r.GET("/playground", gin.WrapH(playground.Handler("GraphQL Playground", "/graphql")))

	// SSE handler
	r.GET("/api/events", sse.HandleSSEConnections)

	// Remote CLI handlers
	r.GET("/describe/odigos", services.DescribeOdigos)
	r.GET("/describe/source/namespace/:namespace/kind/:kind/name/:name", services.DescribeSource)

	return r, nil
}

func httpFileServerWith404(fs http.FileSystem) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fs.Open(r.URL.Path)
		if err != nil {
			// If file not found, serve .html of it (example: /choose-sources -> /choose-sources.html)
			r.URL.Path = r.URL.Path + ".html"
		}
		_, err = fs.Open(r.URL.Path)
		if err != nil {
			// If .html file not found, this route does not exist at all (404) so we should redirect to default
			r.URL.Path = "/"
		}
		http.FileServer(fs).ServeHTTP(w, r)
	})
}

func startWatchers(ctx context.Context, flags *Flags) error {
	err := watchers.StartInstrumentationConfigWatcher(ctx, "")
	if err != nil {
		return fmt.Errorf("error starting InstrumentationConfig watcher: %v", err)
	}

	err = watchers.StartDestinationWatcher(ctx, flags.Namespace)
	if err != nil {
		return fmt.Errorf("error starting Destination watcher: %v", err)
	}

	err = watchers.StartInstrumentationInstanceWatcher(ctx, "")
	if err != nil {
		return fmt.Errorf("error starting InstrumentationInstance watcher: %v", err)
	}

	return nil
}

func main() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := parseFlags()

	if flags.Version {
		fmt.Printf("version.Info{Version:'%s', GitCommit:'%s', BuildDate:'%s'}\n", version.OdigosVersion, version.OdigosCommit, version.OdigosDate)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	defer func() {
		signal.Stop(ch)
		cancel()
	}()

	go common.StartPprofServer(ctx, logr.FromSlogHandler(slog.Default().Handler()))

	// Load destinations data
	err := destinations.Load()
	if err != nil {
		log.Fatalf("Error loading destinations data: %s", err)
	}

	// Connect to Kubernetes
	err = initKubernetesClient(&flags)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %s", err)
	}

	odigosMetrics := collectormetrics.NewOdigosMetrics()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		odigosMetrics.Run(ctx, flags.Namespace)
	}()

	// Start server
	r, err := startHTTPServer(&flags, odigosMetrics)
	if err != nil {
		log.Fatalf("Error starting server: %s", err)
	}

	// Start watchers
	err = startWatchers(ctx, &flags)
	if err != nil {
		log.Fatalf("Error starting watchers: %s", err)
	}

	log.Printf("Odigos UI is available at: http://%s:%d", flags.Address, flags.Port)
	go func() {
		err = r.Run(fmt.Sprintf("%s:%d", flags.Address, flags.Port))
		if err != nil {
			log.Fatalf("Error starting server: %s", err)
		}
	}()

	<-ch
	log.Println("Shutting down Odigos UI...")
	cancel()
	wg.Wait()
}
