# gRPC Config Tests (Collector)

> 23 nodes · cohesion 0.12

## Key Concepts

- **Kind** (9 connections) — `pkg/autodetect/kind.go`
- **.Detect()** (7 connections) — `pkg/autodetect/eks.go`
- **kind.go** (7 connections) — `pkg/autodetect/kind.go`
- **aksDetector** (3 connections) — `pkg/autodetect/aks.go`
- **eksDetector** (3 connections) — `pkg/autodetect/eks.go`
- **gkeDetector** (3 connections) — `pkg/autodetect/gke.go`
- **k3sDetector** (3 connections) — `pkg/autodetect/k3s.go`
- **GetK8SClusterDetails()** (3 connections) — `pkg/autodetect/kind.go`
- **getKindFromDetectors()** (3 connections) — `pkg/autodetect/kind.go`
- **kindDetector** (3 connections) — `pkg/autodetect/kindkind.go`
- **minikubeDetector** (3 connections) — `pkg/autodetect/minikube.go`
- **openshiftDetector** (3 connections) — `pkg/autodetect/openshift.go`
- **getServerVersion()** (2 connections) — `pkg/autodetect/kind.go`
- **ClusterDetails** (1 connections) — `pkg/autodetect/kind.go`
- **ClusterKindDetector** (1 connections) — `pkg/autodetect/kind.go`
- **DetectionArguments** (1 connections) — `pkg/autodetect/kind.go`
- **aks.go** (1 connections) — `pkg/autodetect/aks.go`
- **eks.go** (1 connections) — `pkg/autodetect/eks.go`
- **gke.go** (1 connections) — `pkg/autodetect/gke.go`
- **k3s.go** (1 connections) — `pkg/autodetect/k3s.go`
- **kindkind.go** (1 connections) — `pkg/autodetect/kindkind.go`
- **minikube.go** (1 connections) — `pkg/autodetect/minikube.go`
- **openshift.go** (1 connections) — `pkg/autodetect/openshift.go`

## Relationships

- [[Cluster Kind Detector (eks/gke/aks)]] (62 shared connections)

## Source Files

- `pkg/autodetect/aks.go`
- `pkg/autodetect/eks.go`
- `pkg/autodetect/gke.go`
- `pkg/autodetect/k3s.go`
- `pkg/autodetect/kind.go`
- `pkg/autodetect/kindkind.go`
- `pkg/autodetect/minikube.go`
- `pkg/autodetect/openshift.go`

## Audit Trail

- EXTRACTED: 62 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*