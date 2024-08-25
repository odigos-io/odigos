#!/bin/bash

files=(
  "assert-runtime-detected.yaml"
  "assert-instrumented-and-pipeline.yaml"
)

for file in "${files[@]}"; do
  echo "Asserting $file..."
  # Your logic to assert the YAML file
  # For example, use yamllint or a custom validation
  yamllint "$file" || exit 1
done

echo "All YAML files passed assertions."
