# Instrumentor

The role of this component is to mark pods and workloads for instrumentation by the
odiglet daemonset on each pod's node.

Instrumentation can be done in 2 ways:
1. native SDK - via device manager attached to each pod
2. eBPF SDK - via annotation on the workload object (deployment, daemonset, statefulset).

Instrumentation cue is given on objects if they fulfill the following criteria:
1. The workload object is annotated with `odigos.io/instrument: "true"` or it is not annotated and the namespace is annotated with `odigos.io/instrument: "true"`.
2. There is a runtime details object, set prior by an odiglet that inspected a living pod runtime. The runtime details are crucial for programming language information which is used to determine the correct SDK to use for instrumentation.
3. The collectors are ready to receive telemetry. This is set by the `scheduler` controller.

## Development

To run instrumentor from code:

1. Make sure you have a running k8s cluster with a compatible version of odigos installed.

2. Disable the instrumentor deployment in the cluster:
```sh
$ kubectl scale deployment odigos-instrumentor --replicas=0 -n odigos-system
```

3. Run the instrumentor from code:
```sh
$ go run .
```

