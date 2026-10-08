#!/bin/sh
set -eu

proto=atethat/v1/identity.proto
for service in api core; do
  output="services/$service/internal/gen"
  import_path="github.com/architsangal/AteThat/services/$service/internal/gen/atethat/v1"
  mkdir -p "$output"
  protoc -I contracts/proto \
    --plugin=protoc-gen-go="$(go env GOPATH)/bin/protoc-gen-go" \
    --plugin=protoc-gen-go-grpc="$(go env GOPATH)/bin/protoc-gen-go-grpc" \
    --go_out="$output" --go_opt=paths=source_relative --go_opt="M$proto=$import_path" \
    --go-grpc_out="$output" --go-grpc_opt=paths=source_relative --go-grpc_opt="M$proto=$import_path" \
    "contracts/proto/$proto"
done
