#!/bin/bash
set -e

echo "Generating Protobuf stubs..."
buf generate

echo "Moving generated Go models to api/v1/..."
rm -rf ./api/v1
mkdir -p ./api/v1
mv ./gen/go/api/proto/v1/* ./api/v1/ 2>/dev/null || true
rm -rf ./gen

# Note: openapiv2 plugin outputs to api/swagger (configurable in buf.gen.yaml)
echo "Code generation complete!"
