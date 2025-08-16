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

	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/destinations"
	"github.com/odigos-io/odigos/frontend/graph"
	"github.com/odigos-io/odigos/frontend/graph/loaders"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/kube/watchers"
	"github.com/odigos-io/odigos/frontend/middlewares"
	"github.com/odigos-io/odigos/frontend/services"
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
	"github.com/odigos-io/odigos/frontend/services/db"
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

// The above should point to the UI production build.
// If it's red for you...
// 1. Go to "frontend/webapp/"
// 2. Then run: "yarn install && yarn build"
// After the build completed, there should be a "frontend/webapp/out/" dir (which is ignored from git), that should resolve the red error.

func parseFlags() Flags {
	defaultKubeConfig := env.GetDefaultKubeConfigPath()

	var flags Flags
	flag.BoolVar(&flags.Version, "version", false, "Print Odigos UI version.")
	flag.StringVar(&flags.Address, "address", "localhost", "Address to listen on")
	flag.IntVar(&flags.Port, "port", defaultPort, "Port to listen on")
	flag.BoolVar(&flags.Debug, "debug", false, "Enable debug mode")
	flag.StringVar(&flags.KubeConfig, "kubeconfig", defaultKubeConfig, "Path to kubeconfig file")
	flag.StringVar(&flags.KubeContext, "kube-context", "", "Name of the kubeconfig context to use")
	flag.StringVar(&flags.Namespace, "namespace", env.GetCurrentNamespace(), "Kubernetes namespace where Odigos is installed")
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

func startWatchers(ctx context.Context) error {
	err := watchers.StartInstrumentationConfigWatcher(ctx, "")
	if err != nil {
		return fmt.Errorf("error starting InstrumentationConfig watcher: %v", err)
	}
	return nil
}

func startDatabase() error {
	// Resolve data directory: allow override via env, else use OS-specific config dir
	dataDir := os.Getenv("ODIGOS_UI_DATA_DIR")
	if dataDir == "" {
		// Default to user config dir: ~/.config/odigos or platform equivalent
		if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
			dataDir = cfgDir + "/odigos"
		} else {
			// Fallback to home directory
			if home, err := os.UserHomeDir(); err == nil && home != "" {
				dataDir = home + "/.odigos"
			} else {
				dataDir = "." // ultimate fallback to current dir
			}
		}
	}

	// Ensure directory exists
	if mkErr := os.MkdirAll(dataDir, 0o755); mkErr != nil {
		log.Println("failed to create data directory:", mkErr)
	}

	databasePath := dataDir + "/data.db"
	database, err := db.NewSQLiteDB(databasePath)

	if err != nil {
		// TODO: Move to fatal once db required
		// return err
		log.Println(err, "Failed to connect to DB")
	} else {
		defer database.Close()
		db.InitializeDatabaseSchema(database.GetDB())
	}

	return nil
}

// Serve React app (if page not found serve index.html)
func serveClientFiles(ctx context.Context, r *gin.Engine, dist fs.FS) {
	r.NoRoute(func(c *gin.Context) {
		// Apply OIDC middleware only for routes serving the frontend (GraphQL & Apollo cannot redirect)
		middlewares.OidcMiddleware(ctx)(c)
		if c.IsAborted() {
			return
		}

		fs := http.FS(dist)
		path := c.Request.URL.Path

		_, err := fs.Open(path)
		if err != nil {
			// If file not found, serve .html of it (example: /choose-sources -> /choose-sources.html)
			path += ".html"
		}
		_, err = fs.Open(path)
		if err != nil {
			// If .html file not found, this route does not exist at all (404) so we should redirect to default
			path = "/"
		}

		c.Request.URL.Path = path
		http.FileServer(fs).ServeHTTP(c.Writer, c.Request)
	})
}

func startHTTPServer(ctx context.Context, flags *Flags, logger logr.Logger, odigosMetrics *collectormetrics.OdigosMetricsConsumer) (*gin.Engine, error) {
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

	// Add security headers middleware
	r.Use(middlewares.SecurityHeadersMiddleware)

	// Add CSRF protection middleware
	r.Use(middlewares.CSRFMiddleware())

	// Readiness and Liveness probes
	r.GET("/readyz", func(c *gin.Context) {
		if kube.DefaultClient == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	r.GET("/healthz", func(c *gin.Context) {
		if kube.DefaultClient == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not healthy"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// CSRF token endpoint
	r.GET("/auth/csrf-token", middlewares.CSRFTokenHandler())
	// OIDC/OAuth2 handlers
	r.GET("/auth/oidc-callback", func(c *gin.Context) { services.OidcAuthCallback(ctx, c) })

	gqlExecutableSchema := graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			MetricsConsumer: odigosMetrics,
			Logger:          logger,
		},
	})
	gqlExecutor := executor.New(gqlExecutableSchema)

	// GraphQL handlers
	r.POST("/graphql", func(c *gin.Context) {
		c.Request = c.Request.WithContext(loaders.WithLoaders(c.Request.Context(), loaders.NewLoaders()))
		graph.GetGQLHandler(c.Request.Context(), gqlExecutableSchema).ServeHTTP(c.Writer, c.Request)
	})
	r.GET("/playground", gin.WrapH(playground.Handler("GraphQL Playground", "/graphql")))
	// SSE handler
	r.GET("/api/events", sse.HandleSSEConnections)

	// Remote CLI handlers
	r.POST("/token/update", services.UpdateToken)
	r.GET("/describe/odigos", services.DescribeOdigos)
	r.GET("/describe/source/namespace/:namespace/kind/:kind/name/:name", services.DescribeSource)
	r.GET("/workload", func(c *gin.Context) {
		services.DescribeWorkload(c, gqlExecutor, nil)
	})
	r.GET("/workload/overview", func(c *gin.Context) {
		verbosity := "overview"
		services.DescribeWorkload(c, gqlExecutor, &verbosity)
	})
	r.GET("/workload/health-summary", func(c *gin.Context) {
		verbosity := "healthSummary"
		services.DescribeWorkload(c, gqlExecutor, &verbosity)
	})
	r.GET("/workload/:namespace", func(c *gin.Context) {
		services.DescribeWorkload(c, gqlExecutor, nil)
	})
	r.GET("/workload/:namespace/:kind/:name", func(c *gin.Context) {
		services.DescribeWorkload(c, gqlExecutor, nil)
	})

	r.POST("/source/namespace/:namespace/kind/:kind/name/:name", services.CreateSourceWithAPI)
	r.DELETE("/source/namespace/:namespace/kind/:kind/name/:name", services.DeleteSourceWithAPI)

	return r, nil
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

	logger := logr.FromSlogHandler(slog.Default().Handler())
	go common.StartPprofServer(ctx, logger)

	// Load destinations data
	err := destinations.Load()
	if err != nil {
		log.Fatalf("Error loading destinations data: %s", err)
	}

	// Start SQLite database
	err = startDatabase()
	if err != nil {
		log.Fatalf("Error starting database: %s", err)
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

	// Start watchers
	err = startWatchers(ctx)
	if err != nil {
		log.Fatalf("Error starting watchers: %s", err)
	}

	// Start server
	r, err := startHTTPServer(ctx, &flags, logger, odigosMetrics)
	if err != nil {
		log.Fatalf("Error starting server: %s", err)
	}

	// Serve client (react/next app)
	dist, err := fs.Sub(uiFS, "webapp/out")
	if err != nil {
		log.Fatalf("Error reading webapp/out directory: %s", err)
	}
	serveClientFiles(ctx, r, dist)

	go func() {
		log.Printf("Odigos UI is available at: http://%s:%d", flags.Address, flags.Port)
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
