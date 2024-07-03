# OpAMP Server

This is an implementation of an OpAmp (almost) compatible server.

## Architecture

The server is meant to receive statuses from processes instrumented by odigos, and save the data into InstrumentationInstance CRD to expose it to the frontend or via kubectl.

It resides in its own module, but is intended to run as part of odiglet, as a deamonset on each node, the reasons are:

- since it requires opening a network connection, we want to keep communication local to avoid network policies that might block the connection in some clusters.
- the instrumentation device id injected by odigos is only available on the node itself, and cannot be resolved from a different node in the cluster.

## Development

Odiglet build process will build the server as a binary, and inject it into the odiglet image.
You can make changes to the code of `opampserver`, `agent-nodejs` and test it with `make deploy-odiglet` in the repo root.
Deployments will not restart when odiglet is deployed, so you will need to `kubectl rollout restart deployment {deployment-name}` to see the changes.
