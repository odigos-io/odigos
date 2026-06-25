# Browser instrumentation demo / e2e

A self-contained demo that opts an nginx-served front-end into Odigos browser instrumentation and
verifies that the `odigos-browser-proxy` sidecar (and its iptables-redirect init container) is
injected into the workload's pods.

## Why it isn't in the default e2e matrix

This scenario needs the `odigos-browser-proxy` image present in the cluster, which the default e2e
job does not build/load. Keep it out of `.github/workflows/e2e.yaml`'s `test-scenario` matrix unless
that image is wired into the e2e build/load step.

## Run it locally (kind)

```bash
# from the repo root, with a kind cluster + Odigos installed
make build-browser-proxy load-to-kind-browser-proxy TAG=e2e-test

# build + load the browser agent bundle into the node so the sidecar can serve it
#   (from the opentelemetry-browser repo): make deploy-dev

# run the chainsaw scenario
cd tests/e2e/browser-instrumentation
chainsaw test .
```

## Manual verification of trace export

1. Port-forward the front-end service: `kubectl -n browser-demo port-forward svc/frontend 8080:80`.
2. Open `http://localhost:8080` in a browser and click **Ping**.
3. View source / network tab: the HTML contains an injected `window.__ODIGOS__` config script and a
   `<script src="/__odigos/agent.js">` tag, and the browser POSTs OTLP to `/__odigos/v1/traces`.
4. Confirm browser spans (document load, fetch) arrive at your configured Odigos destination.
