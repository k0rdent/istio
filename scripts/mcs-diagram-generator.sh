#!/usr/bin/env bash

set -euo pipefail

CHART_DIR=${1:-"../charts/k0rdent-istio"}
OUTPUT_FILE_PATH=${2:-"../dev/mcs-dependencies-diagram.md"}
YQ="${3:-yq}"

echo "Generating MCS Dependencies Diagram..."

echo "\`\`\`mermaid" > "$OUTPUT_FILE_PATH"
echo "graph TD" >> "$OUTPUT_FILE_PATH" 

helm template k0rdent-istio "$CHART_DIR" | \
yq e -r 'select(.kind == "MultiClusterService") | [.metadata.name, .spec.dependsOn[]?] | @tsv' - | \
while IFS=$'\t' read -r name deps; do
  if [ -n "$deps" ]; then
    for dep in $deps; do
        echo "  $name -->|dependsOn| $dep" >> "$OUTPUT_FILE_PATH"
    done
  else
    if [ "$name" != "" ]; then
      echo "  $name" >> "$OUTPUT_FILE_PATH"
    fi
  fi
done

echo "\`\`\`" >> "$OUTPUT_FILE_PATH"

echo "Mermaid diagram generated in $OUTPUT_FILE_PATH"