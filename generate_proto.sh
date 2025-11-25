#!/bin/bash

# Generate protobuf code
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/mining.proto

echo "Protobuf code generated successfully"
