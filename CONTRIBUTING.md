# Contributing Guide

- [Contributing Guide](#contributing-guide)
  - [Ways to Contribute](#ways-to-contribute)
  - [Find an Issue](#find-an-issue)
  - [Ask for Help](#ask-for-help)
  - [Local Development](#local-development)
    - [Run Odigos Cli from code](#run-odigos-cli-from-code)
    - [How to Develop Odigos Locally](#how-to-develop-odigos-locally)
    - [How to Build and run Odigos Frontend Locally](#how-to-build-and-run-odigos-frontend-locally)
  - [Odiglet](#odiglet)
    - [builder base image](#builder-base-image)
    - [Remote debugging](#remote-debugging)

Welcome! We are glad that you want to contribute to our project! 💖

As you get started, you are in the best position to give us feedback on areas of
our project that we need help with, including:

- Problems found during setting up a new developer environment
- Gaps in our Quickstart Guide or documentation
- Bugs in our automation scripts

If anything doesn't make sense, or doesn't work when you run it, please open a
bug report and let us know!

## Ways to Contribute

We welcome many different types of contributions, including:

- New features
- Builds, CI/CD
- Bug fixes
- Documentation
- Issue Triage
- Answering questions on Slack/Mailing List
- Web design
- Communications / Social Media / Blog Posts
- Release management

## Find an Issue

We have good first issues for new contributors and help wanted issues suitable
for any contributor. [good first issue](https://github.com/odigos-io/odigos/labels/good%20first%20issue) has extra information to
help you make your first contribution. [help wanted](https://github.com/odigos-io/odigos/labels/help%20wanted) are issues
suitable for someone who isn't a core maintainer and is good to move onto after
your first pull request.

Sometimes there won’t be any issues with these labels. That’s ok! There is
likely still something for you to work on.

Once you see an issue that you'd like to work on, please post a comment saying
that you want to work on it. Something like "I want to work on this" is fine.

## Ask for Help

The best way to reach us with a question when contributing is to ask on:

- The original github issue
- The developer mailing list
- Our Slack channel

## Local Development

This section describes how to setup your local development environment
and test your code changes.

First, follow the [Quickstart Guide](https://docs.odigos.io/quickstart/introduction) in odigos docs to create a local k8s development cluster with a demo application and a functioning odigos installation.

Make sure you are able to:

- [x] run Odigos CLI in your terminal.
- [x] open the demo application UI in your browser to interact with it.
- [x] install odigos in your development cluster with `odigos install`.
- [x] open Odigos UI in your browser to interact with it.
- [x] see telemetry data that odigos generates, for example traces in jaeger.

After you have a working odigos setup, you can start making changes to the code and test them locally.

### Run Odigos Cli from code

The code for the odigos cli tool is found in the `cli` directory [here](https://github.com/odigos-io/odigos/tree/main/cli).
Test your cli code changes by running `go run .` from the `cli` directory:

```bash
go run cli/main.go
```

To run `odigos install` cli command from a local source, you will need to supply a version flag to tell odigos which image tags to install:

```bash
go run cli/main.go install --version v0.1.81
# Installing Odigos version v0.1.81 in namespace odigos-system ...
```

If you test changes to the `install` command, you will need to `go run cli/main.go uninstall` first before you can run install again.

### How to Develop Odigos Locally

The main steps involved when debugging Odigos locally are:

- Use a Kind kubernetes cluster
- Build custom images of Odigos and load them into Kind via:

```bash
TAG=<CURRENT-ODIGOS-VERSION> make build-images load-to-kind
```

- Ensure the TAG matches the Odigos version output from: `odigos version`
- Restart all pods in the `odigos-system` namespace:

```bash
kubectl delete pods --all -n odigos-system
```

See the [Odigos docs](https://docs.odigos.io/intro) for the full steps on debugging Odigos locally.

### How to Build and run Odigos Frontend Locally

Build the frontend

```bash
cd frontend/webapp 
yarn install
yarn build
yarn dev
cd ../.. # back to root of the project for next steps
```

Then run the web server

```bash
cd frontend
go build -o odigos-backend && ./odigos-backend --port 8085 --debug --address 0.0.0.0
```

## Odiglet

### builder base image

Odiglet Dockerfile uses a base image for the builder, which saves up lots of time during builds. The Dockerfile for the base image can be found in `./odiglet/base.Dockerfile` and is consumed like so: `FROM keyval/odiglet-base:v1.0 as builder`
If you need to add additional packages to the build, update this file. Then publish the new base image to dockerhub with the github action named `Publish Odiglet Base Builder` in the `Actions` tab.
You will need to specify the new image tag as a version in the format `v1.0`.
After the image is published, update the dependency in `./odiglet/Dockerfile` to use the new image tag.

### Remote debugging

First, you will have to find which version of Odigos you are running. You can do this by running `odigos version` in your terminal.
Then, run the following command to build Odiglet in debug mode and restart the Odiglet pod:

```bash
TAG=<CURRENT-ODIGOS-VERSION> make debug-odiglet
```

Then, you can attach a debugger to the Odiglet pod. For example, if you are using Goland, you can follow the instructions [here](https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html#step-3-create-the-remote-run-debug-configuration-on-the-client-computer) to attach to a remote process.
For Visual Studio Code, you can use the `.vscode/launch.json` file in this repo to attach to the Odiglet pod.
