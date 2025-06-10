#!/bin/bash

# Default namespace
NAMESPACE=${1:-"odigos-test"}

print_error() {
    printf "\033[31mERROR: %s\033[0m\n" "$1"
}

print_success() {
    printf "\033[32mSUCCESS: %s\033[0m\n" "$1"
}

check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "$1 is not installed"
        exit 1
    fi
}

check_command kubectl
check_command jq

# 1. Verify that all Odigos CRDs are removed
echo "Checking for Odigos CRDs..."
ODIGOS_CRDS=$(kubectl get crd | grep odigos || true)

if [ ! -z "$ODIGOS_CRDS" ]; then
    print_error "Found Odigos CRDs that were not removed:"
    echo "$ODIGOS_CRDS"
    exit 1
else
    print_success "No Odigos CRDs found"
fi

# 2. Verify that all pods don't have Odigos modifications
echo "Checking for pods with Odigos modifications..."

PODS_WITH_ODIGOS=$(kubectl get pods -A -o json | jq -r '
    .items[] | 
    select(.metadata.namespace != "'"$NAMESPACE"'") |
    select(
        any(.metadata.labels | keys[]; startswith("odigos.io/")) or
        any((.spec.affinity // {}).nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[]?.matchExpressions[]?; 
            .key == "odigos.io/odiglet-oss-installed")
    ) | 
    .metadata.namespace + "/" + .metadata.name' 2>/dev/null || true)

if [ ! -z "$PODS_WITH_ODIGOS" ]; then
    print_error "Found pods with Odigos modifications:"
    echo "$PODS_WITH_ODIGOS"
    exit 1
else
    print_success "No pods with Odigos modifications found"
fi

PODS_WITH_ODIGOS_ENV=$(kubectl get pods -A -o json | jq -r '
    .items[] | 
    select(.metadata.namespace != "'"$NAMESPACE"'") |
    select(.spec.containers[].env != null) |
    select(any(.spec.containers[].env[]; .name | startswith("ODIGOS_"))) |
    .metadata.namespace + "/" + .metadata.name' 2>/dev/null || true)

if [ ! -z "$PODS_WITH_ODIGOS_ENV" ]; then
    print_error "Found pods with Odigos environment variables:"
    echo "$PODS_WITH_ODIGOS_ENV"
    exit 1
else
    print_success "No pods with Odigos environment variables found"
fi

PODS_WITH_ODIGOS_RESOURCES=$(kubectl get pods -A -o json | jq -r '
    .items[] | 
    select(.metadata.namespace != "'"$NAMESPACE"'") |
    select(.spec.containers[]?.resources != null) |
    select(.spec.containers[]?.resources.limits != null or .spec.containers[]?.resources.requests != null) |
    select(any(.spec.containers[]; 
        any((.resources.limits // {}) | keys[]; startswith("instrumentation.odigos.io/")) or
        any((.resources.requests // {}) | keys[]; startswith("instrumentation.odigos.io/"))
    )) |
    .metadata.namespace + "/" + .metadata.name' 2>/dev/null || true)

if [ ! -z "$PODS_WITH_ODIGOS_RESOURCES" ]; then
    print_error "Found pods with Odigos resources:"
    echo "$PODS_WITH_ODIGOS_RESOURCES"
    exit 1
else
    print_success "No pods with Odigos resources found"
fi

# 3. Verify no resources remain in the namespace
echo "Checking for remaining resources in namespace $NAMESPACE..."

PODS_IN_NAMESPACE=$(kubectl get pods -n $NAMESPACE 2>/dev/null || true)
if [ ! -z "$PODS_IN_NAMESPACE" ]; then
    print_error "Found pods in namespace $NAMESPACE:"
    echo "$PODS_IN_NAMESPACE"
    exit 1
fi

# the kube-root-ca.crt configmap is created by Kubernetes.
# Helm does not remove the Odigos ns and hence the configmap is not removed.
CONFIGMAPS_IN_NAMESPACE=$(kubectl get configmaps -n $NAMESPACE -o json | jq -r '
    .items[] |
    select(.metadata.name != "kube-root-ca.crt") |
    .metadata.name' 2>/dev/null || true)
if [ ! -z "$CONFIGMAPS_IN_NAMESPACE" ]; then
    print_error "Found configmaps in namespace $NAMESPACE:"
    echo "$CONFIGMAPS_IN_NAMESPACE"
    exit 1
fi


# "default-token" is created in each namespace by k8s prior to k8s 1.24: https://github.com/kubernetes/kubernetes/pull/108309 - so we exclude it
SECRETS_IN_NAMESPACE=$(kubectl get secrets -n $NAMESPACE -o json | jq -r '
    .items[] |
    select(.metadata.name | startswith("default-token") | not) |
    .metadata.name' 2>/dev/null || true)
if [ ! -z "$SECRETS_IN_NAMESPACE" ]; then
    print_error "Found secrets in namespace $NAMESPACE:"
    echo "$SECRETS_IN_NAMESPACE"
    exit 1
fi

# 4. Verify RBAC resources are deleted
echo "Checking for RBAC resources..."

CLUSTER_ROLES=$(kubectl get clusterroles -o name | grep odigos || true)
if [ ! -z "$CLUSTER_ROLES" ]; then
    print_error "Found ClusterRoles related to Odigos:"
    echo "$CLUSTER_ROLES"
    exit 1
else
    print_success "No ClusterRoles related to Odigos found"
fi

CLUSTER_ROLE_BINDINGS=$(kubectl get clusterrolebindings -o name | grep odigos || true)
if [ ! -z "$CLUSTER_ROLE_BINDINGS" ]; then
    print_error "Found ClusterRoleBindings related to Odigos:"
    echo "$CLUSTER_ROLE_BINDINGS"
    exit 1
else
    print_success "No ClusterRoleBindings related to Odigos found"
fi

ROLE_BINDINGS=$(kubectl get rolebindings -n $NAMESPACE -o name | grep odigos || true)
if [ ! -z "$ROLE_BINDINGS" ]; then
    print_error "Found RoleBindings in namespace $NAMESPACE related to Odigos:"
    echo "$ROLE_BINDINGS"
    exit 1
else
    print_success "No RoleBindings related to Odigos found in $NAMESPACE"
fi

ROLES=$(kubectl get roles -n $NAMESPACE -o name | grep odigos || true)
if [ ! -z "$ROLES" ]; then
    print_error "Found Roles in namespace $NAMESPACE related to Odigos:"
    echo "$ROLES"
    exit 1
else
    print_success "No Roles related to Odigos found in $NAMESPACE"
fi

print_success "Odigos has been successfully uninstalled!"