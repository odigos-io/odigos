# AGENTS.md

## Code Guidelines

### Scope of changes

- Keep changes minimal and scoped to the task.
- Do not include unrelated edits, formatting changes, or cleanup.
- Follow existing code patterns and structure.
- Prefer consistency with surrounding code over stylistic changes.
- Do not introduce unnecessary abstractions.
- Generated files are read-only. Never hand-edit them.
- If the task is unclear or not descriptive enough, ask for clarification before making changes.
- Do not introduce new implementations when equivalent functionality already exists in the repository or its dependencies. Search for and reuse existing utilities, helpers, or patterns — including those provided by external libraries already in use. Extend or adapt existing code instead of duplicating functionality.

### Comments

- Do not add comments that restate the code.
- Prefer clearer code over explanatory comments.
- Add comments only when they provide necessary context, explain non-obvious behavior, or constraints.
- When adding comments that fit the above, place them above the relevant code line in the function if they explain logic specific to that line, or above the struct field if the comment is describing a field.

### Common packages

- `common` package is not k8s specific and can be consumed by projects not running on k8s (e.g VMs) - it shouldn't have any k8s dependency.
- `k8sutils` is the common package for k8s related utilities, such as event filters, odigos custom resources utilities and common k8s objects related functions.

## References

When repository files do not fully answer a question, prefer these references:

- [Kubernetes Documentation](https://kubernetes.io/docs/home/)
- [Kubebuilder book](https://book.kubebuilder.io/)
- [OpenTelemetry documentation and specifications](https://opentelemetry.io/docs/)
- [Effective go](https://go.dev/doc/effective_go)

