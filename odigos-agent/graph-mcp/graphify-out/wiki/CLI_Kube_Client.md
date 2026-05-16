# CLI Kube Client

> 25 nodes · cohesion 0.15

## Key Concepts

- **agents.go** (10 connections) — `pkg/instrumentation/fs/agents.go`
- **CopyAgentsDirectoryToHost()** (8 connections) — `pkg/instrumentation/fs/agents.go`
- **copy.go** (8 connections) — `pkg/instrumentation/fs/copy.go`
- **copyDirectories()** (7 connections) — `pkg/instrumentation/fs/copy.go`
- **removeChangedFilesFromKeepMap()** (5 connections) — `pkg/instrumentation/fs/agents.go`
- **createTestFiles()** (5 connections) — `pkg/instrumentation/fs/copy_test.go`
- **copy_test.go** (4 connections) — `pkg/instrumentation/fs/copy_test.go`
- **.Create()** (4 connections) — `pkg/kube/runtime_details/instrumentationconfigs_controller.go`
- **calculateFileHash()** (3 connections) — `pkg/instrumentation/fs/agents.go`
- **renameFileWithHashSuffix()** (3 connections) — `pkg/instrumentation/fs/agents.go`
- **writeKeeplist()** (3 connections) — `pkg/instrumentation/fs/agents.go`
- **copyFile()** (3 connections) — `pkg/instrumentation/fs/copy.go`
- **createDotnetDeprecatedDirectories()** (3 connections) — `pkg/instrumentation/fs/copy.go`
- **getFiles()** (3 connections) — `pkg/instrumentation/fs/copy.go`
- **BenchmarkCopyDirectories()** (3 connections) — `pkg/instrumentation/fs/copy_test.go`
- **TestCopyDirectories()** (3 connections) — `pkg/instrumentation/fs/copy_test.go`
- **TestGetFiles()** (3 connections) — `pkg/instrumentation/fs/copy_test.go`
- **worker()** (3 connections) — `pkg/instrumentation/fs/copy.go`
- **findExistingHashVersionFiles()** (2 connections) — `pkg/instrumentation/fs/agents.go`
- **generateRenamedFilePath()** (2 connections) — `pkg/instrumentation/fs/agents.go`
- **isDirEmptyOrNotExist()** (2 connections) — `pkg/instrumentation/fs/agents.go`
- **runSingleRsyncSync()** (2 connections) — `pkg/instrumentation/fs/agents.go`
- **getArch()** (2 connections) — `pkg/instrumentation/fs/copy.go`
- **getNumberOfWorkers()** (2 connections) — `pkg/instrumentation/fs/copy.go`
- **HostContainsEbpfDir()** (1 connections) — `pkg/instrumentation/fs/copy.go`

## Relationships

- [[Instrumentor Manager]] (90 shared connections)
- [[Odiglet File Copy]] (2 shared connections)
- [[Source Object Docs]] (1 shared connections)
- [[VM Agent Docs]] (1 shared connections)

## Source Files

- `pkg/instrumentation/fs/agents.go`
- `pkg/instrumentation/fs/copy.go`
- `pkg/instrumentation/fs/copy_test.go`
- `pkg/kube/runtime_details/instrumentationconfigs_controller.go`

## Audit Trail

- EXTRACTED: 76 (81%)
- INFERRED: 18 (19%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*