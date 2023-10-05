#!/bin/bash
set -eux

protoc \
  --go_out=./ \
  --go-grpc_out=./ \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  --go_opt=Mprofile2.proto=github.com/andreimatei/delve-agent/agentrpc \
  --go-grpc_opt=Mprofile2.proto=github.com/andreimatei/delve-agent/agentrpc \
  rpc.proto profile2.proto
  
 
#  -I=agentrpc \
#  --go-grpc_opt=Mprofile.proto=github.com/andreimatei/delve-agent/agentrpc \
# --go-grpc_opt=Mprofile.proto=github.com/andreimatei/delve-agent/agentrpc \
