#!/bin/sh
set -eu

for service in api core; do
  output="services/$service/internal/gen"
  import_path="github.com/architsangal/AteThat/services/$service/internal/gen/atethat/v1"
  mkdir -p "$output"
  protoc -I contracts/proto \
    --plugin=protoc-gen-go="$(go env GOPATH)/bin/protoc-gen-go" \
    --plugin=protoc-gen-go-grpc="$(go env GOPATH)/bin/protoc-gen-go-grpc" \
    --go_out="$output" --go_opt=paths=source_relative \
    --go_opt="Matethat/v1/identity.proto=$import_path" --go_opt="Matethat/v1/targets.proto=$import_path" --go_opt="Matethat/v1/foods.proto=$import_path" --go_opt="Matethat/v1/logs.proto=$import_path" --go_opt="Matethat/v1/dashboard.proto=$import_path" --go_opt="Matethat/v1/health.proto=$import_path" --go_opt="Matethat/v1/analysis.proto=$import_path" \
    --go-grpc_out="$output" --go-grpc_opt=paths=source_relative \
    --go-grpc_opt="Matethat/v1/identity.proto=$import_path" --go-grpc_opt="Matethat/v1/targets.proto=$import_path" --go-grpc_opt="Matethat/v1/foods.proto=$import_path" --go-grpc_opt="Matethat/v1/logs.proto=$import_path" --go-grpc_opt="Matethat/v1/dashboard.proto=$import_path" --go-grpc_opt="Matethat/v1/health.proto=$import_path" --go-grpc_opt="Matethat/v1/analysis.proto=$import_path" \
    contracts/proto/atethat/v1/identity.proto contracts/proto/atethat/v1/targets.proto contracts/proto/atethat/v1/foods.proto contracts/proto/atethat/v1/logs.proto contracts/proto/atethat/v1/dashboard.proto contracts/proto/atethat/v1/health.proto contracts/proto/atethat/v1/analysis.proto
done
