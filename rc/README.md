# Odigos Release Candidate Helm Charts

This directory contains release candidate versions of Odigos Helm charts.

## Usage

To add this repository for release candidates:

```bash
helm repo add odigos-rc https://odigos-io.github.io/odigos/rc/
helm repo update
```

## Installing Release Candidates

```bash
helm install odigos odigos-rc/odigos --version <rc-version>
```

## Warning

⚠️ **Release candidates are pre-release versions and may contain bugs or incomplete features.**
⚠️ **Do not use in production environments.**

For stable releases, use the main repository:
```bash
helm repo add odigos https://odigos-io.github.io/odigos/
```
