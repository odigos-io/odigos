#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║     🦀 Rust Binary Detection Test (Stripped vs Unstripped)      ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""

echo "=== Step 1: Building Rust test images ==="
docker build --target unstripped -t rust-detection-test:unstripped .
docker build --target stripped -t rust-detection-test:stripped .

echo ""
echo "=== Step 2: Analyzing binaries locally ==="
CONTAINER_ID=$(docker create rust-detection-test:unstripped)
docker cp "$CONTAINER_ID:/app/server" /tmp/rust-unstripped
docker rm "$CONTAINER_ID"

CONTAINER_ID=$(docker create rust-detection-test:stripped)
docker cp "$CONTAINER_ID:/app/server" /tmp/rust-stripped
docker rm "$CONTAINER_ID"

echo "--- Unstripped binary ---"
echo "Size: $(ls -lh /tmp/rust-unstripped | awk '{print $5}')"
echo "Symbols:"
if nm /tmp/rust-unstripped 2>/dev/null | grep -E '__rust_|rust_begin_unwind|_ZN4core|_ZN3std' | head -5; then
    echo "✅ Rust symbols found"
else
    echo "⚠️ No Rust symbols (might be on different arch)"
fi

echo ""
echo "--- Stripped binary ---"
echo "Size: $(ls -lh /tmp/rust-stripped | awk '{print $5}')"
echo "Symbols:"
if nm /tmp/rust-stripped 2>/dev/null | grep -E '__rust_|rust_begin_unwind|_ZN4core|_ZN3std' | head -5; then
    echo "✅ Rust symbols found (unexpected for stripped)"
else
    echo "✅ No symbols (expected for stripped binary)"
fi

echo ""
echo "Panic strings in stripped binary:"
if strings /tmp/rust-stripped | grep -E 'panicked at|/rustc/|unwrap\(\)' | head -3; then
    echo "✅ Panic strings found - detection will work!"
else
    echo "❌ No panic strings found"
fi

echo ""
echo "=== Step 3: Loading images into Kind ==="
kind load docker-image rust-detection-test:unstripped --name odigos-dev
kind load docker-image rust-detection-test:stripped --name odigos-dev

echo ""
echo "=== Step 4: Deploying test applications ==="
kubectl apply -f k8s-manifests.yaml
kubectl rollout status deployment/rust-unstripped -n rust-detection-test --timeout=120s
kubectl rollout status deployment/rust-stripped -n rust-detection-test --timeout=120s

echo ""
echo "=== Step 5: Verifying pods are running ==="
kubectl get pods -n rust-detection-test

echo ""
echo "=== Step 6: Checking language detection ==="
sleep 5

echo ""
echo "--- Unstripped app detection ---"
UNSTRIPPED_IC=$(kubectl get instrumentationconfig -n rust-detection-test deployment-rust-unstripped -o yaml 2>/dev/null || echo "Not found")
if echo "$UNSTRIPPED_IC" | grep -q "language: rust"; then
    echo "✅ Unstripped app detected as Rust"
else
    echo "⚠️ Unstripped app not yet detected (Odigos may need to rescan)"
    echo "   Current status:"
    kubectl get instrumentationconfig -n rust-detection-test 2>/dev/null || echo "   No instrumentationconfigs yet"
fi

echo ""
echo "--- Stripped app detection ---"
STRIPPED_IC=$(kubectl get instrumentationconfig -n rust-detection-test deployment-rust-stripped -o yaml 2>/dev/null || echo "Not found")
if echo "$STRIPPED_IC" | grep -q "language: rust"; then
    echo "✅ Stripped app detected as Rust"
else
    echo "⚠️ Stripped app not yet detected (Odigos may need to rescan)"
fi

echo ""
echo "=== Test Complete ==="
echo ""
echo "To manually trigger Odigos re-detection:"
echo "  kubectl rollout restart deployment/rust-unstripped -n rust-detection-test"
echo "  kubectl rollout restart deployment/rust-stripped -n rust-detection-test"
echo ""
echo "To check detection status:"
echo "  kubectl get instrumentationconfig -n rust-detection-test -o yaml"
echo ""
echo "To cleanup:"
echo "  kubectl delete namespace rust-detection-test"

