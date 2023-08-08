#!/bin/bash
set -eux

protoc \
  --go_out=./agentrpc \
  --go-grpc_out=./agentrpc \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  -I=agentrpc \
  --go_opt=Mprofile.proto=github.com/andreimatei/delve-agent/agentrpc \
  --go-grpc_opt=Mprofile.proto=github.com/andreimatei/delve-agent/agentrpc \
  agentrpc/rpc.proto agentrpc/profile.proto
