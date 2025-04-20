# Odigos End to End Testing

In addition to unit tests, Odigos has a suite of end-to-end tests that are run on every pull request.
These tests are installing multiple microservices, instrument with Odigos, generate traffic, and validate the results.

## Tools

- [Kubernetes In Docker (Kind)](https://kind.sigs.k8s.io/) - a tool for running local Kubernetes clusters using Docker container “nodes”.
- [Chainsaw](https://kyverno.github.io/chainsaw/) - To orchestrate the different Kubernetes actions.

## Running e2e locally

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

- yq and jq installed:

```bash
brew install yq
brew install jq
```

### Preparing

You can run all the below steps with `make dev-tests-setup`.

- Fresh Kubernetes cluster in kubectl context. For local development, you can use KinD but also managed clusters like EKS should work. you can create the cluster with `make dev-tests-kind-cluster`.
- Odigos CLI compiled at the `cli` directory in odigos OSS repo (which is expected to be cloned as sibling of the current repo). To compile the cli executable, go to the OSS repository and run: `make cli-build`.
- Odigos component images (instrumentor, autoscaler, odiglet etc) tagged with `e2e-test` preloaded to the cluster. If you are using KinD you can run: `TAG=e2e-test make build-images load-to-kind`.

### Running the Tests

To run specific scenarios, for example `multi-apps` run from Odigos root directory:

```bash
chainsaw test tests/e2e/multi-apps
```

## Writing new scenarios

Every scenario should include some/all of the following:

- Install destination (Odigos "simple-trace-db" deployment)
- Install test applications
- Install Odigos
- Select apps for instrumentation and configure destination
- Generate traffic
- Validate traces

Scenarios are written in yaml files called `chainsaw-test.yaml` according to the Chainsaw schema.

See the [following document](https://kyverno.github.io/chainsaw/latest/test/) for more information on how to write scenarios.

Scenarios should be placed in the `tests/e2e/<scenraio-name>` directory and Trace validations should be placed in the `tests/e2e/<scenraio-name>/queries` directory.

After writing and testing new scenario, you should also add it to the GitHub Action file location at:
`.github/workflows/e2e.yaml` to run it on every pull request.

## Working with simple-trace-db

"simple-trace-db" is an in-memory database that stores spans and allow querying them via a simple API and common query language.
It is used to execute JMESPath queries to assert the traces that odigos generates.

### Connecting to simple-trace-db

In order to run trace queries, you need to connect to simple-trace-db.
The db is installed automatically in the e2e test, so if you ran a scenario you can connect to it.
You can do this by port-forwarding the service:

```bash
kubectl port-forward svc/simple-trace-db 4318:4318 -n traces
```

### Querying traces

Then you can execute simple-trace-db queries via:

```bash
curl -G -s http://localhost:4318/v1/traces --data-urlencode "jmespath=length([?span.serviceName == 'frontend']) > \`0\`"
```

Getting individual trace by trace id is not yet supported by simple-trace-db.

For all APIs it is recommended to pipe the results to `jq` for better readability and `less` for paging.

### Writing queries

See examples on how to write queries for `simple-trace-db` in the [playground](https://github.com/odigos-io/simple-trace-db/tree/main/playground) of the DB. To run it in test, create the following yaml.

```yaml
apiVersion: e2e.tests.odigos.io/v1
kind: TraceTest
description: <Description>
query: |
    simple-trace-db query
expected:
  count: <Number of expected spans>
```

Once you have the file, you can run the test via:

```bash
tests/e2e/common/simple_trace_db_query_runner.sh <path-to-yaml-file>
```
