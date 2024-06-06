# OpAMP Server

This is an implementation of an OpAmp (almost) compatible server.

## Architecture

The server is meant to receive statuses from processes instrumented by odigos, and save the data into InstrumentationInstance CRD to expose it to the frontend or via kubectl.

It resides in its own module, but is intended to run as part of odiglet, as a deamonset on each node, the reasons are:

- since it requires opening a network connection, we want to keep communication local to avoid network policies that might block the connection in some clusters.
- the instrumentation device id injected by odigos is only available on the node itself, and cannot be resolved from a different node in the cluster.

## Development

For development, it is recommended to run the OpAMP server as a standalone process, and not as part of odiglet. This makes it so it can run on MacOS, and reduces the complexity that odiglet brings.

`make dev` will run the server from the source code, watch for changes in the source code, and restart the server when changes are detected. to use it, you need to install [`nodemon`](https://www.npmjs.com/package/nodemon) tool (`npm install -g nodemon`).

