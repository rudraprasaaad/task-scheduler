#!/bin/bash

# Create output directory
mkdir -p internal/grpc/generated

# Generate Go code from proto files
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/task/task.proto

protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/worker/worker.proto

echo "Protocol buffer code generated successfully!"
