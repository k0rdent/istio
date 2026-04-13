#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

echo "Running chart smoke tests"

helm dependency update "charts/k0rdent-istio"
helm lint --strict "charts/k0rdent-istio"

# Utility manifests chart: only propagation.{enabled,data}; schema requires non-empty data when enabled.
helm lint --strict "charts/k0rdent-istio-manifests"

helm lint --strict "charts/k0rdent-istio-manifests" \
  --set propagation.enabled=true \
  --set-string propagation.data=$'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test'

if helm lint --strict "charts/k0rdent-istio-manifests" \
  --set propagation.enabled=true \
  --set propagation.data= >/dev/null 2>&1; then
  echo "Expected manifests chart lint to fail when propagation is enabled but data is empty"
  exit 1
fi

if helm template smoke "charts/k0rdent-istio-manifests" | grep -q '[^[:space:]]'; then
  echo "Expected default manifests render to be whitespace-only (propagation disabled)"
  exit 1
fi

ns_render="$(mktemp)"
gw_render="$(mktemp)"
prop_render="$(mktemp)"
main_render="$(mktemp)"

helm template smoke "charts/k0rdent-istio-manifests" \
  --set propagation.enabled=true \
  --set-string propagation.data=$'apiVersion: v1\nkind: Namespace\nmetadata:\n  name: istio-system\n  annotations:\n    helm.sh/resource-policy: keep\n  labels:\n    topology.istio.io/network: alpha-network' >"$ns_render"

grep -q -- "kind: Namespace" "$ns_render"
grep -q -- "name: istio-system" "$ns_render"
grep -q -- "topology.istio.io/network: alpha-network" "$ns_render"

helm template smoke "charts/k0rdent-istio-manifests" \
  --set propagation.enabled=true \
  --set-string propagation.data=$'apiVersion: networking.istio.io/v1\nkind: Gateway\nmetadata:\n  name: eastwest\n  namespace: istio-system\nspec:\n  servers:\n    - hosts:\n        - "*.edge-a.local"' >"$gw_render"

grep -q -- "kind: Gateway" "$gw_render"
grep -Fq -- "*.edge-a.local" "$gw_render"

helm template smoke "charts/k0rdent-istio-manifests" \
  --set propagation.enabled=true \
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
grep -Fq -- 'copy "Data" | nindent 14 }}' "$main_render"

rm -f "$ns_render" "$gw_render" "$prop_render" "$main_render"

echo "Chart smoke tests passed"
