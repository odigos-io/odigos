# Contributing Guide

- [New Contributor Guide](#contributing-guide)
  - [Ways to Contribute](#ways-to-contribute)
  - [Find an Issue](#find-an-issue)
  - [Ask for Help](#ask-for-help)
  - [Local Development](#local-development)
    - [Run Odigos Cli from code](#run-odigos-cli-from-code)

Welcome! We are glad that you want to contribute to our project! 💖

As you get started, you are in the best position to give us feedback on areas of
our project that we need help with including:

- Problems found during setting up a new developer environment
- Gaps in our Quickstart Guide or documentation
- Bugs in our automation scripts

If anything doesn't make sense, or doesn't work when you run it, please open a
bug report and let us know!

## Ways to Contribute

We welcome many different types of contributions including:

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
for any contributor. [good first issue](https://github.com/keyval-dev/odigos/labels/good%20first%20issue) has extra information to
help you make your first contribution. [help wanted](https://github.com/keyval-dev/odigos/labels/help%20wanted) are issues
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

This sections describe how to setup your local development environment
and test your code changes.

First, follow the [Quickstart Guide](https://docs.odigos.io/intro) in odigos docs to create a k8s cluster with demo application.

### Run Odigos Cli from code

The code for the odigos cli tool is found at the `cli` directory [here](https://github.com/keyval-dev/odigos/tree/main/cli).
Test your cli code changes by running `go run .` from the `cli` directory:

```bash
➜  odigos git:(main) cd cli/
➜  cli git:(main) go run .       
```

To run `odigos install` cli command from local source, you will need to supply a version flag to tell odigos which image tags to install:
```bash
➜  cli git:(main) go run . install --version v0.1.81
Installing Odigos version v0.1.81 in namespace odigos-system ...
```
