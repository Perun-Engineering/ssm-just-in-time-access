#!/bin/bash
set -e

# Build script for SSM Access Manager Lambda functions

echo "Building Lambda functions for arm64..."

# Create bin directory
mkdir -p bin

# Build each Lambda function
echo "Building request-handler..."
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bin/bootstrap cmd/request-handler/main.go
cd bin && zip request-handler.zip bootstrap && rm bootstrap && cd ..

echo "Building approval-handler..."
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bin/bootstrap cmd/approval-handler/main.go
cd bin && zip approval-handler.zip bootstrap && rm bootstrap && cd ..

echo "Building document-creator..."
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bin/bootstrap cmd/document-creator/main.go
cd bin && zip document-creator.zip bootstrap && rm bootstrap && cd ..

echo "Building expiration-cleanup..."
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bin/bootstrap cmd/expiration-cleanup/main.go
cd bin && zip expiration-cleanup.zip bootstrap && rm bootstrap && cd ..

echo "Building admin-handler..."
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bin/bootstrap cmd/admin-handler/main.go
cd bin && zip admin-handler.zip bootstrap && rm bootstrap && cd ..

echo "Building admin-slack-handler..."
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bin/bootstrap cmd/admin-slack-handler/main.go
cd bin && zip admin-slack-handler.zip bootstrap && rm bootstrap && cd ..

echo "Build complete! Lambda packages are in bin/"
