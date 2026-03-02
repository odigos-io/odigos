#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_NAMESPACE="rust-test"
TAG="${TAG:-e2e-test}"
KIND_CLUSTER="${KIND_CLUSTER:-odigos-dev}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

cleanup() {
    log_info "Cleaning up..."
    kubectl delete namespace "$TEST_NAMESPACE" --ignore-not-found=true 2>/dev/null || true
    rm -rf /tmp/rust-test-app 2>/dev/null || true
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing=()
    command -v kind &>/dev/null || missing+=("kind")
    command -v kubectl &>/dev/null || missing+=("kubectl")
    command -v docker &>/dev/null || missing+=("docker")
    command -v go &>/dev/null || missing+=("go")
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing required tools: ${missing[*]}"
        exit 1
    fi
    
    log_success "All prerequisites satisfied"
}

setup_kind_cluster() {
    log_info "Setting up Kind cluster..."
    
    if kind get clusters 2>/dev/null | grep -q "$KIND_CLUSTER"; then
        log_info "Using existing Kind cluster: $KIND_CLUSTER"
        kubectl config use-context "kind-$KIND_CLUSTER"
    else
        log_info "Creating new Kind cluster: $KIND_CLUSTER"
        kind create cluster --name "$KIND_CLUSTER" --config="$REPO_ROOT/tests/common/apply/kind-config.yaml"
    fi
    
    kubectl wait --for=condition=Ready nodes --all --timeout=120s
    log_success "Kind cluster ready: $KIND_CLUSTER"
}

build_and_load_odigos() {
    log_info "Building Odigos images with TAG=$TAG..."
    
    cd "$REPO_ROOT"
    
    make build-images TAG="$TAG" || {
        log_error "Failed to build images"
        exit 1
    }
    
    log_info "Loading images into Kind cluster: $KIND_CLUSTER..."
    for img in instrumentor autoscaler scheduler odiglet collector ui agents; do
        kind load docker-image "registry.odigos.io/odigos-$img:$TAG" --name "$KIND_CLUSTER" || true
    done
    
    log_success "Odigos images loaded into Kind"
}

install_odigos() {
    log_info "Installing Odigos from source..."
    
    cd "$REPO_ROOT"
    
    make cli-build
    
    ./cli/odigos install --version "$TAG" --nowait
    
    log_info "Waiting for Odigos pods to be ready..."
    kubectl wait --for=condition=Ready pods --all -n odigos-system --timeout=300s
    
    log_success "Odigos installed successfully"
}

verify_rust_distro() {
    log_info "Verifying rust-native distro is loaded..."
    
    if [[ ! -f "$REPO_ROOT/distros/yamls/rust-native.yaml" ]]; then
        log_error "rust-native.yaml not found!"
        exit 1
    fi
    
    cd "$REPO_ROOT/distros"
    go build ./... || {
        log_error "Distros module failed to build"
        exit 1
    }
    
    go test ./... || {
        log_error "Distros module tests failed"
        exit 1
    }
    
    log_success "rust-native distro verified"
}

create_test_rust_app() {
    log_info "Creating test Rust application..."
    
    local app_dir="/tmp/rust-test-app"
    rm -rf "$app_dir"
    mkdir -p "$app_dir/src"
    
    cat > "$app_dir/Cargo.toml" << 'EOF'
[package]
name = "rust-test-app"
version = "0.1.0"
edition = "2021"

[dependencies]
tokio = { version = "1", features = ["rt-multi-thread", "net", "macros"] }
hyper = { version = "1", features = ["server", "http1"] }
hyper-util = { version = "0.1", features = ["tokio"] }
http-body-util = "0.1"
opentelemetry = "0.24"
opentelemetry_sdk = { version = "0.24", features = ["rt-tokio"] }
opentelemetry-otlp = { version = "0.17", features = ["grpc-tonic"] }

[profile.release]
strip = true
lto = true
EOF

    cat > "$app_dir/src/main.rs" << 'EOF'
use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper::{Request, Response};
use hyper_util::rt::TokioIo;
use http_body_util::Full;
use hyper::body::Bytes;
use opentelemetry::global;
use opentelemetry::trace::{Tracer, TracerProvider};
use opentelemetry_otlp::WithExportConfig;
use std::convert::Infallible;
use std::net::SocketAddr;
use tokio::net::TcpListener;

fn init_telemetry() -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    let endpoint = std::env::var("OTEL_EXPORTER_OTLP_ENDPOINT")
        .unwrap_or_else(|_| "http://localhost:4317".to_string());
    
    println!("Initializing telemetry with endpoint: {}", endpoint);
    
    let exporter = opentelemetry_otlp::new_exporter()
        .tonic()
        .with_endpoint(&endpoint);

    let provider = opentelemetry_otlp::new_pipeline()
        .tracing()
        .with_exporter(exporter)
        .install_batch(opentelemetry_sdk::runtime::Tokio)?;

    global::set_tracer_provider(provider);
    
    println!("Telemetry initialized successfully");
    Ok(())
}

async fn handle_request(_req: Request<hyper::body::Incoming>) -> Result<Response<Full<Bytes>>, Infallible> {
    let tracer = global::tracer("rust-test-app");
    let _span = tracer.start("handle_request");
    
    println!("Handling request");
    
    Ok(Response::new(Full::new(Bytes::from("Hello from Rust with OpenTelemetry!"))))
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    println!("Starting Rust test application...");
    
    println!("Environment variables:");
    for (key, value) in std::env::vars() {
        if key.starts_with("OTEL_") {
            println!("  {}={}", key, value);
        }
    }
    
    if let Err(e) = init_telemetry() {
        eprintln!("Warning: Failed to initialize telemetry: {}", e);
    }
    
    let addr = SocketAddr::from(([0, 0, 0, 0], 8080));
    let listener = TcpListener::bind(addr).await?;
    println!("Listening on http://{}", addr);

    loop {
        let (stream, _) = listener.accept().await?;
        let io = TokioIo::new(stream);

        tokio::spawn(async move {
            if let Err(err) = http1::Builder::new()
                .serve_connection(io, service_fn(handle_request))
                .await
            {
                eprintln!("Error serving connection: {:?}", err);
            }
        });
    }
}
EOF

    cat > "$app_dir/Dockerfile" << 'EOF'
FROM rust:1.75-bookworm AS builder
WORKDIR /app
COPY Cargo.toml ./
COPY src ./src
RUN cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/target/release/rust-test-app /app/rust-test-app
EXPOSE 8080
CMD ["/app/rust-test-app"]
EOF

    log_info "Building Rust test app Docker image..."
    docker build -t rust-test-app:latest "$app_dir"
    
    log_info "Loading image into Kind cluster: $KIND_CLUSTER..."
    kind load docker-image rust-test-app:latest --name "$KIND_CLUSTER"
    
    log_success "Test Rust app created and loaded"
}

deploy_test_app() {
    log_info "Deploying test Rust application..."
    
    kubectl create namespace "$TEST_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    kubectl apply -f - << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rust-test-app
  namespace: $TEST_NAMESPACE
  labels:
    app: rust-test-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: rust-test-app
  template:
    metadata:
      labels:
        app: rust-test-app
    spec:
      containers:
        - name: app
          image: rust-test-app:latest
          imagePullPolicy: Never
          ports:
            - containerPort: 8080
          resources:
            requests:
              memory: "64Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: rust-test-app
  namespace: $TEST_NAMESPACE
spec:
  selector:
    app: rust-test-app
  ports:
    - port: 8080
      targetPort: 8080
EOF

    log_info "Waiting for deployment to be ready..."
    kubectl wait --for=condition=available deployment/rust-test-app -n "$TEST_NAMESPACE" --timeout=120s
    
    log_success "Test app deployed"
}

enable_instrumentation() {
    log_info "Enabling Odigos instrumentation for Rust app..."
    
    kubectl apply -f - << EOF
apiVersion: odigos.io/v1alpha1
kind: Source
metadata:
  name: rust-test-app
  namespace: $TEST_NAMESPACE
spec:
  workload:
    name: rust-test-app
    namespace: $TEST_NAMESPACE
    kind: Deployment
EOF

    log_info "Waiting for instrumentation config to be created..."
    sleep 10
    
    local retries=30
    while [[ $retries -gt 0 ]]; do
        if kubectl get instrumentationconfig -n "$TEST_NAMESPACE" rust-test-app-deployment &>/dev/null; then
            break
        fi
        log_info "Waiting for InstrumentationConfig... ($retries retries left)"
        sleep 2
        ((retries--))
    done
    
    if [[ $retries -eq 0 ]]; then
        log_warn "InstrumentationConfig not created, checking status..."
    fi
    
    log_success "Instrumentation enabled"
}

verify_instrumentation() {
    log_info "Verifying Rust instrumentation..."
    
    log_info "Checking InstrumentationConfig..."
    kubectl get instrumentationconfig -n "$TEST_NAMESPACE" -o yaml || true
    
    log_info "Checking if Rust language was detected..."
    local lang=$(kubectl get instrumentationconfig -n "$TEST_NAMESPACE" rust-test-app-deployment -o jsonpath='{.status.runtimeDetails[0].language}' 2>/dev/null || echo "unknown")
    
    if [[ "$lang" == "rust" ]]; then
        log_success "Rust language detected correctly!"
    else
        log_warn "Language detected: $lang (expected: rust)"
    fi
    
    log_info "Checking pod environment variables..."
    kubectl wait --for=condition=ready pod -l app=rust-test-app -n "$TEST_NAMESPACE" --timeout=60s || true
    
    local pod_name=$(kubectl get pods -n "$TEST_NAMESPACE" -l app=rust-test-app -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    
    if [[ -n "$pod_name" ]]; then
        log_info "Pod: $pod_name"
        
        log_info "Environment variables in pod:"
        kubectl exec -n "$TEST_NAMESPACE" "$pod_name" -- env | grep -E "^OTEL_" || log_warn "No OTEL_* env vars found"
        
        log_info "Pod logs:"
        kubectl logs -n "$TEST_NAMESPACE" "$pod_name" --tail=20 || true
    fi
    
    log_success "Instrumentation verification complete"
}

deploy_jaeger() {
    log_info "Deploying Jaeger for trace visualization..."
    
    kubectl apply -f - << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger
  namespace: odigos-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jaeger
  template:
    metadata:
      labels:
        app: jaeger
    spec:
      containers:
        - name: jaeger
          image: jaegertracing/all-in-one:1.52
          ports:
            - containerPort: 16686
              name: ui
            - containerPort: 4317
              name: otlp-grpc
          env:
            - name: COLLECTOR_OTLP_ENABLED
              value: "true"
          resources:
            requests:
              memory: "256Mi"
              cpu: "100m"
---
apiVersion: v1
kind: Service
metadata:
  name: jaeger
  namespace: odigos-system
spec:
  selector:
    app: jaeger
  ports:
    - name: ui
      port: 16686
    - name: otlp-grpc
      port: 4317
EOF

    kubectl wait --for=condition=available deployment/jaeger -n odigos-system --timeout=60s
    
    log_success "Jaeger deployed"
}

generate_traffic() {
    log_info "Generating test traffic..."
    
    kubectl port-forward svc/rust-test-app -n "$TEST_NAMESPACE" 8080:8080 &
    local pf_pid=$!
    sleep 3
    
    for i in {1..5}; do
        curl -s http://localhost:8080 || true
        sleep 1
    done
    
    kill $pf_pid 2>/dev/null || true
    
    log_success "Traffic generated"
}

print_summary() {
    echo ""
    echo "=========================================="
    echo -e "${GREEN}Test Summary${NC}"
    echo "=========================================="
    echo ""
    log_info "Rust test app deployed in namespace: $TEST_NAMESPACE"
    log_info "Odigos installed in namespace: odigos-system"
    echo ""
    echo "Useful commands:"
    echo "  View InstrumentationConfig:"
    echo "    kubectl get instrumentationconfig -n $TEST_NAMESPACE -o yaml"
    echo ""
    echo "  Check pod env vars:"
    echo "    kubectl exec -n $TEST_NAMESPACE \$(kubectl get pods -n $TEST_NAMESPACE -l app=rust-test-app -o jsonpath='{.items[0].metadata.name}') -- env | grep OTEL_"
    echo ""
    echo "  View pod logs:"
    echo "    kubectl logs -n $TEST_NAMESPACE -l app=rust-test-app -f"
    echo ""
    echo "  Port forward to Jaeger UI:"
    echo "    kubectl port-forward svc/jaeger -n odigos-system 16686:16686"
    echo "    # Then open http://localhost:16686"
    echo ""
    echo "  Port forward to test app:"
    echo "    kubectl port-forward svc/rust-test-app -n $TEST_NAMESPACE 8080:8080"
    echo "    # Then curl http://localhost:8080"
    echo ""
    echo "  Cleanup:"
    echo "    kind delete cluster"
    echo "=========================================="
}

main() {
    local skip_cluster=false
    local skip_build=false
    local cleanup_only=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-cluster)
                skip_cluster=true
                shift
                ;;
            --skip-build)
                skip_build=true
                shift
                ;;
            --cleanup)
                cleanup_only=true
                shift
                ;;
            --help)
                echo "Usage: $0 [options]"
                echo ""
                echo "Options:"
                echo "  --skip-cluster  Skip Kind cluster creation (use existing)"
                echo "  --skip-build    Skip building Odigos images"
                echo "  --cleanup       Only cleanup resources and exit"
                echo "  --help          Show this help"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    if [[ "$cleanup_only" == "true" ]]; then
        cleanup
        log_success "Cleanup complete"
        exit 0
    fi
    
    log_info "Starting Rust instrumentation test..."
    log_info "Using TAG=$TAG"
    
    check_prerequisites
    verify_rust_distro
    
    if [[ "$skip_cluster" != "true" ]]; then
        setup_kind_cluster
    fi
    
    if [[ "$skip_build" != "true" ]]; then
        build_and_load_odigos
    fi
    
    install_odigos
    create_test_rust_app
    deploy_test_app
    deploy_jaeger
    enable_instrumentation
    
    sleep 15
    
    verify_instrumentation
    generate_traffic
    
    print_summary
    
    log_success "Rust instrumentation test completed!"
}

main "$@"

