#!/bin/bash
#
# Generate all beehive protobuf bindings.
# Run from repository root.
#
set -e

# directories containing protos to be built
DIRS="./metapb ./rpcpb ./apipb"

PRJ_PB_PATH="${GOPATH}/src/github.com/deepfabric/busybee/pkg/pb"

for dir in ${DIRS}; do
	pushd ${dir}
		protoc  -I=.:"${PRJ_PB_PATH}":"${GOPATH}/src" --gogofaster_out=plugins=grpc:.  *.proto
		goimports -w *.pb.go
	popd
done
