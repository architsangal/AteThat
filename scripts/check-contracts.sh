#!/bin/sh
set -eu

ruby scripts/check-openapi.rb contracts/openapi/v1.yaml
descriptor="$(mktemp)"
trap 'rm -f "$descriptor"' EXIT HUP INT TERM
protoc --proto_path=contracts/proto --descriptor_set_out="$descriptor" contracts/proto/atethat/v1/identity.proto
echo "Protocol Buffer descriptor valid"
