#!/bin/bash

deploy_curl_pod() {
  local pod_name="temp-curl-checker"
  local namespace="${1:-default}"
  echo "Creating temporary pod..."
  kubectl run "$pod_name" -n "$namespace" --image=curlimages/curl --restart=Never --command -- sleep 3600 >/dev/null

  echo "Waiting for pod $pod_name to be ready..."
  kubectl wait --for=condition=Ready pod/"$pod_name" -n "$namespace" --timeout=60s
}

delete_curl_pod() {
  local pod_name="temp-curl-checker"
  local namespace="${1:-default}"
  echo "Deleteing temporary pod..."
  kubectl delete pod "$pod_name" -n "$namespace" --ignore-not-found >/dev/null
}

run_curl_cmd() {
  local pod_name="temp-curl-checker"
  local namespace="${1:-default}"
  local url="$2"
  kubectl exec -n "$namespace" "$pod_name" -- curl -s $url
}