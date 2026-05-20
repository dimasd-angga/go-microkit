#!/usr/bin/env bash
# Regenerate gRPC stubs. Requires protoc + protoc-gen-go + protoc-gen-go-grpc.
#
#   brew install protobuf
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

for f in proto/user/user.proto proto/order/order.proto; do
    echo ">> $f"
    protoc \
        --proto_path=proto \
        --go_out=. --go_opt=module=github.com/dimasd-angga/go-microkit \
        --go-grpc_out=. --go-grpc_opt=module=github.com/dimasd-angga/go-microkit \
        "$f"
done

echo "done."
