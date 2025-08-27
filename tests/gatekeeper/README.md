This folder includes some common policies or `constraints` that can be configured using `gatekeeper`.
See `install-gatekeeper` make target in the main Makefile for the commands to install gatekeeper and apply the constraints.

- `restrict-host-namespace` will not allow pods that have `hostPid: true` or `hostIPC: true`. It allows to configure a list of `excludedImages`.
- `restrict-hostpath`  will not allow any volume mount of type `hostPath` except a list of allowed paths.
- `restrict-privileged` will not allow privileged containers except a list of allowed ones.

Each file contains a `ConstraintTemplate` which is a CR that `gatekeeper` handles - it creates a template/schema for a policy/constraint.
In order to apply the policy in the cluster, an instance of the generated constraint needs to be created. Hence, each file contains also the instance to be applied for each constraint.
Immediately applying the constraint instance after the template can fail - since it takes time for creating the constraint CRD from the templates - and trying to create the instance before the API server is familiar with the created constraint will fail. Hence, the `install-gatekeeper` target has some retry mechanism.