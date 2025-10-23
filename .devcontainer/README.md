# Devcontainer for Odigos

This setup of devcontainer allows you to quickly get started with Odigos and a local Kubernetes cluster using `kind`.

## Who is this for?
- If you do not want to install anything on your local machine, and want to get a quick setup for Odigos, this is for you.

## Quick Development Setup (only devcontainer)
1. Start the devcontainer (`CMD + Shift + P` -> Reopen in Container)

## Full Odigos Setup (devcontainer + kind cluster)
1. Create kind cluster (can run locally as well)
> This specfic version of kind is required because "latest" is sometimes not compatible with the devcontainer setup, so we pin.
```sh
kind create cluster --name kind --image kindest/node:v1.31.4
```

2. Start the devcontainer (`CMD + Shift + P` -> Reopen in Container)
3. Install odigos CLI (this can be done by following the [documentation](https://docs.odigos.io/quickstart/installation) or by running `make helm-install` )

```sh
make helm-install ODIGOS_CLI_VERSION=1.2.x
# or 
odigos install
```

4. Install Jaeger

```sh
kubectl apply -f https://raw.githubusercontent.com/odigos-io/simple-demo/main/kubernetes/jaeger.yaml
```

5. Deploy the sample application

```sh
kubectl apply -f https://raw.githubusercontent.com/odigos-io/simple-demo/main/kubernetes/deployment.yaml
```

6. Port forward Odigos UI

```sh
odigos ui
```
## (Re)connecting to the `kind` cluster
If you re-started the devcontainer or the kind cluster you might need to reconnect the devcontainer to the kind cluster. You can do so by running the following script from within the devcontainer:
```sh
./.devcontainer/connect-kind.sh
```

## Developers Setup (or: how to plug-and-play your own stuff during development)
> Note: The cluster is using `kind` at the host level and we are using a devcontainer, which are running on the same docker VM, therefore - any changes to the cluster (like installing `odigos` or `jaeger`) need to be applied against the docker daemon at the host; Therefore - we need to make sure that the devcontainer can communicate with the the kubeconfig(`kubectl`), kind, docker daemon, etc. Overall, everything is set up to work out of the box, but if you need to make any special changes to the cluster or other magic - the bugs will probably reside at the above domains.

### Overall deployment flow
The flow of deploying a new version of Odigos to the `kind` cluster is as follows:
1. Build the image(s) at your desired tag
2. Load the tag into kind
3. Update the k8 object to use the new tag
In order to make this easier, we have a Makefile that automates these steps.

### Building, installing and deploying a custom version of Odigos
Following these steps will allow you to build, load and deploy a custom version of Odigos to your `kind` cluster.

1. Install the CLI at your desired version
```sh
make helm-install ODIGOS_CLI_VERSION=1.2.x
```
> At this point, given a non-existing tag, you will see an error when trying to deploy, which is expected.

> If Odigos CLI is already installed, you can skip this step or upgrade it to the desired version:
```sh
make helm-upgrade ODIGOS_CLI_VERSION=1.2.x
```

2. Build Odigos images at your desired tag
```sh
make deploy TAG=<YOUR_DESIRED_TAG> # e.g. TAG=1.2.x
```
By doing this, you will build the image and load it into kind so everything should be set up properly.


### Further granularity
If you need to deploy a specific container, or just the CRDs, or just the UI, you can do so by following these steps:

#### Building and deploying a specific component
```sh
make deploy-ui TAG=<YOUR_DESIRED_TAG>
# Options:
# 1. deploy-instrumentor
# 2. deploy-autoscaler
# 3. deploy-scheduler
# 4. deploy-odiglet
# 5. deploy-collector
# 6. deploy-ui
# 7. deploy-cli
# 8. deploy-agents
```

#### Loading a specific component into kind
```sh
make load-to-kind-% TAG=<YOUR_DESIRED_TAG>
# Options:
# 1. load-to-kind-instrumentor
# 2. load-to-kind-autoscaler
# 3. load-to-kind-scheduler
# 4. load-to-kind-odiglet
# 5. load-to-kind-collector
# 6. load-to-kind-ui
# 7. load-to-kind-cli
# 8. load-to-kind-agents
```

### Deploying CRDs

CRDs are installed via the CLI, so we need to either install it or upgrade it.

```sh
make crd-apply
```
---
💁‍♂️ More options are available in the [Makefile](../Makefile).