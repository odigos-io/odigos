# OpAMP Server

This is an implementation of an OpAmp (almost) compatible server.

## Architecture

The server is meant to receive statuses from processes instrumented by odigos, and save the data into InstrumentationInstance CRD to expose it to the frontend or via kubectl.

It resides in its own module, but is intended to run as part of odiglet, as a deamonset on each node, the reasons are:

- since it requires opening a network connection, we want to keep communication local to the node to keep issues to a minimum (not all clusters allow any pod to communicate with other nodes).
- the instrumentation device id injected by odigos is only available on the node itself, and cannot be resolved from a different node in the cluster.

