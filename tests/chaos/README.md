# Odigos Chaos Testing

Another kind of tests we would like to run, are chaos tests.
The purpose of those tests is to tests the odigos platform with different fault and errors on the cluster level.

## Tools

- [Kubernetes In Docker (Kind)](https://kind.sigs.k8s.io/) - a tool for running local Kubernetes clusters using Docker container “nodes”.
- [Chainsaw](https://kyverno.github.io/chainsaw/) - To orchestrate the different Kubernetes actions.
- [Chaos-mesh](https://github.com/chaos-mesh/chaos-mesh) - In order to simulate faults in the cluster.

## Running chaos tests locally

### Prerequisites

Install these tools once when setting up your local testing environment the first time.

- [Kubernetes In Docker (KinD)](https://kind.sigs.k8s.io/) - a tool for running local Kubernetes clusters using Docker container “nodes”.

- [Chainsaw](https://kyverno.github.io/chainsaw/) - To orchestrate the different Kubernetes actions.
  - Hombrew:

  ```bash
  brew tap kyverno/chainsaw https://github.com/kyverno/chainsaw
  brew install kyverno/chainsaw/chainsaw
  ```

  - Go:

  ```bash
  go install github.com/kyverno/chainsaw@latest
  ```

- helm, yq and jq installed:

```bash
brew install yq
brew install jq
brew install helm
```

### Preparing

You can run all the below steps with `make dev-tests-setup`.

- Fresh Kubernetes cluster in kubectl context. For local development, you can use KinD but also managed clusters like EKS will work. you can create the cluster with `make dev-tests-kind-cluster`.
- Odigos CLI compiled at the `cli` directory in odigos OSS repo (which is expected to be cloned as sibling of the current repo). To compile the cli executable, go to the OSS repository and run: `make cli-build`.

### Running the Tests

To run specific scenarios, for example `network-latency/leader-election` run from Odigos root directory:

```bash
chainsaw test tests/chaos/network-latency/leader-election
```

## Writing new scenarios

Every scenario should include some/all of the following:

- Install destination (`simple-trace-db`)
- Install test applications
- Install Odigos
- Select apps for instrumentation and configure destination
- Generate traffic
- Validate traces

Scenarios are written in yaml files called `chainsaw-test.yaml` according to the Chainsaw schema.

See the [following document](https://kyverno.github.io/chainsaw/latest/test/) for more information on how to write scenarios.

Scenarios should be placed in the `tests/chaos/<fault-created>/<scenario-name>` directory.

After writing and testing new scenario, you should also add it to the GitHub Action file location at:
`.github/workflows/chaos.yaml` to run it on every pull request.
