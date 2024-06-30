# Odiglet

## Development
One of Odiglet's jobs is to manage the different eBPF instrumentations. Loading an eBPF instrumentation requires having compiled eBPF programs (.o files). This compilation is taking place in Odiglet's Dockerfile and it requires the auto instrumentation code. This makes debugging locally on a non-linux system different compared to the other Odigos components.
Assuming a setup with an active kind cluster with Odigos installed:
1. Run `make debug-odiglet` or `TAG=<some_tag> make debug-odiglet` which will build Odiglet in a debug image which includes a Go debugger.
In addition, it will port-forward the debugger port for remote debug.
2. Using vscode launch the `Remote Odiglet` configuration.
3. Debug the code.
