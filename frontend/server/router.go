package server

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"github.com/odigos-io/odigos/frontend/graph"
	"github.com/odigos-io/odigos/frontend/graph/loaders"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/middlewares"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/odigos-io/odigos/frontend/services/sse"
	"github.com/odigos-io/odigos/frontend/webapp"
)

// RouterOpts is everything the caller supplies on top of Deps when building
// the gin engine. The webapp bundle and the /workloads page are served from
// the frontend Go module itself (the webapp and graph packages), so callers
// — OSS and out-of-tree alike — don't supply any embeds.
//
// ExtraMounts is the hook out-of-tree wrappers use to attach additional
// handlers (e.g. the enterprise MCP server) AFTER all OSS routes have been
// registered but BEFORE the NoRoute SPA fallback fires.
type RouterOpts struct {
	ExtraMounts []func(r *gin.Engine, deps *Deps)
}

// BuildRouter constructs the gin engine, registers every OSS HTTP route
// (health, CSRF/OIDC, GraphQL, SSE, the workload describe endpoints, the
// /workloads static page, and /diagnose/download), invokes any ExtraMounts,
// and finally installs the React SPA NoRoute fallback.
func BuildRouter(ctx context.Context, deps *Deps, opts RouterOpts) (*gin.Engine, error) {
	var r *gin.Engine
	if deps.Flags.Debug {
		r = gin.Default()
	} else {
		gin.SetMode(gin.ReleaseMode)
		r = gin.New()
		r.Use(gin.Recovery())
	}

	r.Use(cors.Default())
	r.Use(middlewares.SecurityHeadersMiddleware)
	r.Use(middlewares.CSRFMiddleware())

	// Readiness / Liveness — gated on the default kube client being installed.
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

	// Auth.
	r.GET("/auth/csrf-token", middlewares.CSRFTokenHandler())
	r.GET("/auth/oidc-callback", func(c *gin.Context) { services.OidcAuthCallback(ctx, c) })

	// GraphQL.
	gqlSchema := graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			MetricsConsumer: deps.OdigosMetrics,
			Logger:          deps.Logger,
			PromAPI:         deps.PromAPI,
			K8sCacheClient:  deps.K8sCacheClient,
			ProfileStore:    deps.ProfileStore,
		},
	})
	gqlExecutor := executor.New(gqlSchema)
	r.POST("/graphql", func(c *gin.Context) {
		loader := loaders.NewLoaders(deps.Logger, deps.K8sCacheClient)
		baseCtx := c.Request.Context()
		if c.GetHeader(middlewares.AdminOverrideHeader) == "true" {
			baseCtx = middlewares.WithAdminOverride(baseCtx)
		}
		baseCtx = loaders.WithLoaders(baseCtx, loader)
		c.Request = c.Request.WithContext(baseCtx)
		graph.GetGQLHandler(c.Request.Context(), gqlSchema).ServeHTTP(c.Writer, c.Request)
	})
	r.GET("/playground", gin.WrapH(playground.Handler("GraphQL Playground", "/graphql")))

	// SSE.
	r.GET("/api/events", sse.HandleSSEConnections)

	// Remote CLI handlers.
	r.POST("/token/update", services.UpdateToken)
	r.GET("/describe/odigos", services.DescribeOdigos)
	r.GET("/describe/source/namespace/:namespace/kind/:kind/name/:name", services.DescribeSource)
	r.GET("/workload", func(c *gin.Context) {
		services.DescribeWorkload(c, deps.Logger, gqlExecutor, nil, deps.K8sCacheClient)
	})
	r.GET("/workload/overview", func(c *gin.Context) {
		v := "overview"
		services.DescribeWorkload(c, deps.Logger, gqlExecutor, &v, deps.K8sCacheClient)
	})
	r.GET("/workload/health-summary", func(c *gin.Context) {
		v := "healthSummary"
		services.DescribeWorkload(c, deps.Logger, gqlExecutor, &v, deps.K8sCacheClient)
	})
	r.GET("/workload/:namespace", func(c *gin.Context) {
		services.DescribeWorkload(c, deps.Logger, gqlExecutor, nil, deps.K8sCacheClient)
	})
	r.GET("/workload/:namespace/:kind/:name", func(c *gin.Context) {
		services.DescribeWorkload(c, deps.Logger, gqlExecutor, nil, deps.K8sCacheClient)
	})
	r.GET("/workload/:namespace/:kind/:name/pods", func(c *gin.Context) {
		v := "pods"
		services.DescribeWorkload(c, deps.Logger, gqlExecutor, &v, deps.K8sCacheClient)
	})

	r.POST("/source/namespace/:namespace/kind/:kind/name/:name", services.CreateSourceWithAPI)
	r.DELETE("/source/namespace/:namespace/kind/:kind/name/:name", services.DeleteSourceWithAPI)

	// Workloads static HTML page (embedded in the graph package).
	r.GET("/workloads", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", graph.WorkloadsHTML)
	})

	// Diagnose download endpoint (paired with the GraphQL diagnose mutation).
	r.GET("/diagnose/download", services.DiagnoseDownload)

	// Extra mounts (the enterprise wrapper's hook for /mcp).
	for _, mount := range opts.ExtraMounts {
		if mount != nil {
			mount(r, deps)
		}
	}

	// React SPA fallback — must be last so application routes match first.
	// The bundle is embedded in the frontend/webapp package.
	dist, err := webapp.FS()
	if err != nil {
		return nil, fmt.Errorf("loading embedded webapp bundle: %w", err)
	}
	serveClientFiles(ctx, r, dist)

	return r, nil
}

// serveClientFiles installs the NoRoute SPA fallback: tries the requested
// path, then path+".html", finally "/" so client-side routes work.
func serveClientFiles(ctx context.Context, r *gin.Engine, dist fs.FS) {
	r.NoRoute(gzip.Gzip(gzip.DefaultCompression), func(c *gin.Context) {
		// OIDC middleware applies only on UI-serving routes; GraphQL/Apollo cannot redirect.
		middlewares.OidcMiddleware(ctx)(c)
		if c.IsAborted() {
			return
		}

		hfs := http.FS(dist)
		path := c.Request.URL.Path

		if _, err := hfs.Open(path); err != nil {
			path += ".html"
		}
		if _, err := hfs.Open(path); err != nil {
			path = "/"
		}

		c.Request.URL.Path = path
		http.FileServer(hfs).ServeHTTP(c.Writer, c.Request)
	})
}
