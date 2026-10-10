#!/bin/sh
set -eu

ruby scripts/check-openapi.rb contracts/openapi/v1.yaml
descriptor="$(mktemp)"
trap 'rm -f "$descriptor"' EXIT HUP INT TERM
protoc --proto_path=contracts/proto --descriptor_set_out="$descriptor" contracts/proto/atethat/v1/identity.proto contracts/proto/atethat/v1/targets.proto contracts/proto/atethat/v1/foods.proto contracts/proto/atethat/v1/logs.proto contracts/proto/atethat/v1/dashboard.proto contracts/proto/atethat/v1/health.proto contracts/proto/atethat/v1/analysis.proto
echo "Protocol Buffer descriptor valid"
