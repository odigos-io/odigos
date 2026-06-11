// Package server is the importable bootstrap for the Odigos UI backend. It
// exposes everything needed to stand up the frontend — flag parsing, dependency
// construction, background workers, the gin router, and the serve loop — as a
// small, ordered, public API so out-of-tree binaries (notably the enterprise UI
// image) can reuse the exact setup graph instead of duplicating it.
//
// Lifecycle (call in order):
//
//	flags := server.ParseFlags()
//	deps, err := server.Bootstrap(ctx, flags, logger)   // synchronous setup; no goroutines
//	wg, err := server.StartBackground(ctx, deps)         // long-running workers
//	r, err := server.BuildRouter(ctx, deps, server.RouterOpts{ /* ExtraMounts */ })
//	err = server.ServeAndWait(cancel, deps, r, sigCh, wg)
//
// Bootstrap and StartBackground are deliberately separate so an out-of-tree main
// can interleave its own steps between them (e.g. the enterprise binary verifies
// its on-prem license after the kube client exists but before workers start).
//
// RouterOpts.ExtraMounts is the extension seam: handlers registered there are
// attached AFTER every OSS route but BEFORE the SPA NoRoute fallback, so an
// out-of-tree wrapper can add endpoints (e.g. the enterprise MCP server at /mcp)
// without the SPA catch-all swallowing them. The OSS UI passes no ExtraMounts;
// MCP ships only with the enterprise image.
package server
