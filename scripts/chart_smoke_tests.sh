#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

echo "Running chart smoke tests"

helm dependency update "charts/k0rdent-istio"
helm lint --strict "charts/k0rdent-istio"

helm lint --strict "charts/k0rdent-istio-manifests" \
  --set features.namespace.enabled=true \
  --set clusterName=test

helm lint --strict "charts/k0rdent-istio-manifests" \
  --set features.gateway.enabled=true \
  --set clusterName=test \
  --set gateway.resource.apiVersion=networking.istio.io/v1 \
  --set gateway.resource.kind=Gateway \
  --set gateway.resource.metadata.name=test \
  --set gateway.resource.metadata.namespace=istio-system

helm lint --strict "charts/k0rdent-istio-manifests" \
  --set features.propagation.enabled=true \
  --set-string propagation.data=$'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test'

if helm template smoke "charts/k0rdent-istio-manifests" >/dev/null 2>&1; then
  echo "Expected manifests chart to fail when no features are enabled"
  exit 1
fi

if helm template smoke "charts/k0rdent-istio-manifests" \
  --set features.namespace.enabled=true \
  --set features.gateway.enabled=true \
  --set clusterName=test \
  --set gateway.resource.apiVersion=networking.istio.io/v1 \
  --set gateway.resource.kind=Gateway \
  --set gateway.resource.metadata.name=test \
  --set gateway.resource.metadata.namespace=istio-system >/dev/null 2>&1; then
  echo "Expected manifests chart to fail when multiple features are enabled"
  exit 1
fi

ns_render="$(mktemp)"
gw_render="$(mktemp)"
prop_render="$(mktemp)"
main_render="$(mktemp)"

helm template smoke "charts/k0rdent-istio-manifests" \
  --set features.namespace.enabled=true \
  --set clusterName=alpha >"$ns_render"

grep -q -- "kind: Namespace" "$ns_render"
grep -q -- "name: \"istio-system\"" "$ns_render"
grep -q -- "topology.istio.io/network: \"alpha-network\"" "$ns_render"

helm template smoke "charts/k0rdent-istio-manifests" \
  --set features.gateway.enabled=true \
  --set clusterName=edge-a \
  --set gateway.resource.apiVersion=networking.istio.io/v1 \
  --set gateway.resource.kind=Gateway \
  --set gateway.resource.metadata.name=eastwest \
  --set gateway.resource.metadata.namespace=istio-system \
  --set gateway.resource.spec.servers[0].hosts[0]='*.{clusterName}.local' >"$gw_render"

grep -q -- "kind: Gateway" "$gw_render"
grep -q -- "- '*.edge-a.local'" "$gw_render"

helm template smoke "charts/k0rdent-istio-manifests" \
  --set features.propagation.enabled=true \
  --set-string propagation.data=$'apiVersion: v1\nkind: Secret\nmetadata:\n  name: copied' >"$prop_render"

grep -q -- "kind: Secret" "$prop_render"
grep -q -- "name: copied" "$prop_render"

helm template smoke "charts/k0rdent-istio" >"$main_render"
if grep -q -- "localSourceRef:" "$main_render"; then
  echo "Rendered k0rdent-istio chart still contains localSourceRef"
  exit 1
fi

grep -q -- "template: k0rdent-istio-propagation" "$main_render"
grep -q -- "templateResourceRefs:" "$main_render"
grep -q -- "{{ copy \"Data\" }}" "$main_render"

rm -f "$ns_render" "$gw_render" "$prop_render" "$main_render"

echo "Chart smoke tests passed"
