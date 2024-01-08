# Instrumentor

This component has 2 responsibilities:
1. Add and remove instrumentation devices to pods via the pod template spec.
2. Cleanup the instrumented application objects from the cluster.

## Instrumentation Devices

A workload (Deployment, DaemonSet, StatefulSet) should be instrumented if and only if:

1. The workload object is annotated with `odigos.io/instrument: "true"` or it is not annotated and the namespace is annotated with `odigos.io/instrument: "true"`.
2. There is a runtime details object, set prior by an odiglet that inspected a living pod runtime. The runtime details are crucial for programming language information which is used to determine the correct SDK to use for instrumentation.
3. The collectors are ready to receive telemetry. This is set by the `scheduler` controller.

The "instrumentor" component is responsible for watching the cluster for changes in the above conditions and adding or removing instrumentation devices to the workload.

Downstream, Odiglet is responsible for handling the instrumentation devices and mounting fs volumes to the pod, adding environment variables, and attaching eBPF probes according to the instrumentation devices.

## Delete Instrumented Application Objects

Odiglet is responsible for creating the instrumented application objects once it inspects the runtime details of a pod. Since odiglet is a daemonset, the logic for creating and updating the object is run multiple times, because extracting the runtime details must run on the same node as the pod. 

Deleting the object, however, can only be done once. The instrumentor is responsible for watching for changes in the workload manifests and deleting the instrumented application objects when the workload instrumentation label is removed.

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
