// loadtest battle-tests the netmetrics EndpointResolver + ServiceResolver under load:
// it repeatedly Refresh()es the /proc-wide, per-netns socket table while real
// containers/processes generate heavy connection churn, measuring refresh latency,
// resolution accuracy, error count, goroutines and heap — to surface leaks, panics,
// races, or pathological slowdowns before production. Depends only on the shared
// module (no OBI/eBPF), so it isolates the NEW resolver code.
//
// Env:
//
//	LT_REGISTER   "pid:service,pid:service"  PIDs to map to service names
//	LT_VERIFY     "ip:port,ip:port"          endpoints that MUST resolve every cycle
//	LT_INTERVAL   refresh interval (default 1s)
//	LT_DURATION   total run (default 0 = until SIGINT)
package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/odigos-io/odigos/netmetrics"
)

func main() {
	interval := envDur("LT_INTERVAL", time.Second)
	duration := envDur("LT_DURATION", 0)

	pidSvc := map[int]netmetrics.Service{}
	for _, p := range splitPairs(os.Getenv("LT_REGISTER")) {
		if pid, err := strconv.Atoi(p[0]); err == nil {
			pidSvc[pid] = netmetrics.Service{Name: p[1]}
		}
	}
	verify := splitVerify(os.Getenv("LT_VERIFY"))

	endpoints, err := netmetrics.NewEndpointResolver()
	if err != nil {
		fmt.Println("FATAL: resolver:", err)
		os.Exit(1)
	}
	resolver := netmetrics.NewServiceResolver(endpoints,
		func(pid int) (netmetrics.Service, bool) { s, ok := pidSvc[pid]; return s, ok },
		nil)

	go func() { _ = http.ListenAndServe("localhost:6060", nil) }() // pprof

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if duration > 0 {
		var c context.CancelFunc
		ctx, c = context.WithTimeout(ctx, duration)
		defer c()
	}

	var (
		mu          sync.Mutex
		durs        []time.Duration // refresh durations in current window
		refreshes   atomic.Int64
		refreshErrs atomic.Int64
		verifyMiss  atomic.Int64
		verifyTot   atomic.Int64
		panics      atomic.Int64
	)

	// recover-wrapped refresh: a panic here is a hard failure we must catch & count.
	doRefresh := func() {
		defer func() {
			if r := recover(); r != nil {
				panics.Add(1)
				fmt.Println("PANIC in Refresh:", r)
			}
		}()
		t0 := time.Now()
		if err := endpoints.Refresh(); err != nil {
			refreshErrs.Add(1)
		}
		d := time.Since(t0)
		mu.Lock()
		durs = append(durs, d)
		mu.Unlock()
		refreshes.Add(1)
		// accuracy: every verify endpoint must resolve to a non-IP service name.
		for _, v := range verify {
			verifyTot.Add(1)
			fi, ok := resolver.Resolve("0.0.0.0", 0, v.ip, v.port) // dst-side local lookup
			if !ok || fi.Local.Name == "" || fi.Local.Name == v.ip {
				// try as source too (some services are the connection initiator)
				fi2, ok2 := resolver.Resolve(v.ip, v.port, "0.0.0.0", 0)
				if !ok2 || fi2.Local.Name == "" || fi2.Local.Name == v.ip {
					verifyMiss.Add(1)
				}
			}
		}
	}

	report := time.NewTicker(5 * time.Second)
	defer report.Stop()
	refresh := time.NewTicker(interval)
	defer refresh.Stop()

	start := time.Now()
	fmt.Printf("loadtest started: interval=%s duration=%s registered=%d verify=%d\n",
		interval, duration, len(pidSvc), len(verify))

	for {
		select {
		case <-ctx.Done():
			printSummary(start, refreshes.Load(), refreshErrs.Load(), panics.Load(), verifyTot.Load(), verifyMiss.Load(), endpoints.Size())
			return
		case <-refresh.C:
			doRefresh()
		case <-report.C:
			mu.Lock()
			window := durs
			durs = nil
			mu.Unlock()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			p50, p99, mx := pct(window, 50), pct(window, 99), maxD(window)
			fmt.Printf("[%5.0fs] refresh n=%d/win p50=%-6s p99=%-6s max=%-6s | endpoints=%d | err=%d panic=%d | verify_miss=%d/%d | goroutines=%d heapMB=%.1f\n",
				time.Since(start).Seconds(), len(window), p50, p99, mx,
				endpoints.Size(), refreshErrs.Load(), panics.Load(),
				verifyMiss.Load(), verifyTot.Load(), runtime.NumGoroutine(), float64(m.HeapAlloc)/1e6)
		}
	}
}

func printSummary(start time.Time, n, errs, panics, vtot, vmiss int64, sz int) {
	acc := 100.0
	if vtot > 0 {
		acc = 100 * float64(vtot-vmiss) / float64(vtot)
	}
	fmt.Println("================ LOADTEST SUMMARY ================")
	fmt.Printf("ran=%s refreshes=%d errors=%d panics=%d endpoints=%d\n", time.Since(start).Round(time.Second), n, errs, panics, sz)
	fmt.Printf("verify: %d/%d resolved  (accuracy=%.2f%%)\n", vtot-vmiss, vtot, acc)
	if panics == 0 && errs == 0 {
		fmt.Println("RESULT: PASS (no panics, no refresh errors)")
	} else {
		fmt.Println("RESULT: FAIL")
	}
	fmt.Println("=================================================")
}

type endpoint struct {
	ip   string
	port int
}

func splitPairs(s string) [][2]string {
	var out [][2]string
	for _, p := range strings.Split(s, ",") {
		if kv := strings.SplitN(strings.TrimSpace(p), ":", 2); len(kv) == 2 && kv[0] != "" {
			out = append(out, [2]string{kv[0], kv[1]})
		}
	}
	return out
}
func splitVerify(s string) []endpoint {
	var out []endpoint
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		i := strings.LastIndex(p, ":")
		if i <= 0 {
			continue
		}
		port, err := strconv.Atoi(p[i+1:])
		if err != nil {
			continue
		}
		out = append(out, endpoint{p[:i], port})
	}
	return out
}
func envDur(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
func pct(d []time.Duration, p int) time.Duration {
	if len(d) == 0 {
		return 0
	}
	c := append([]time.Duration(nil), d...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	idx := (p * len(c)) / 100
	if idx >= len(c) {
		idx = len(c) - 1
	}
	return c[idx].Round(time.Millisecond)
}
func maxD(d []time.Duration) time.Duration {
	var m time.Duration
	for _, x := range d {
		if x > m {
			m = x
		}
	}
	return m.Round(time.Millisecond)
}
