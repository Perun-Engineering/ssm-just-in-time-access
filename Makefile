.PHONY: test build clean deploy lint

# Build all Lambda functions
build:
	@echo "Building Lambda functions for arm64..."
	GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bin/bootstrap cmd/request-handler/main.go && cd bin && zip request-handler.zip bootstrap && rm bootstrap && cd ..
	GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bin/bootstrap cmd/approval-handler/main.go && cd bin && zip approval-handler.zip bootstrap && rm bootstrap && cd ..
	GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bin/bootstrap cmd/document-creator/main.go && cd bin && zip document-creator.zip bootstrap && rm bootstrap && cd ..
	GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bin/bootstrap cmd/expiration-cleanup/main.go && cd bin && zip expiration-cleanup.zip bootstrap && rm bootstrap && cd ..
	GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bin/bootstrap cmd/admin-handler/main.go && cd bin && zip admin-handler.zip bootstrap && rm bootstrap && cd ..
	GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bin/bootstrap cmd/admin-slack-handler/main.go && cd bin && zip admin-slack-handler.zip bootstrap && rm bootstrap && cd ..

# Run all tests
test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Run property-based tests
test-property:
	go test -v -race ./test/property/...

# Run integration tests
test-integration:
	go test -v -race ./test/integration/...

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.txt

# Deploy infrastructure
deploy:
	cd infrastructure/terraform && terraform apply

# Deploy Lambda functions only (after build)
deploy-lambdas:
	@echo "Deploying Lambda functions..."
	aws lambda update-function-code --function-name test-ssm-request-handler --zip-file fileb://bin/request-handler.zip --region us-east-1
	aws lambda update-function-code --function-name test-ssm-approval-handler --zip-file fileb://bin/approval-handler.zip --region us-east-1
	aws lambda update-function-code --function-name test-ssm-document-creator --zip-file fileb://bin/document-creator.zip --region us-east-1
	aws lambda update-function-code --function-name test-ssm-expiration-cleanup --zip-file fileb://bin/expiration-cleanup.zip --region us-east-1
	aws lambda update-function-code --function-name test-ssm-admin-handler --zip-file fileb://bin/admin-handler.zip --region us-east-1
	aws lambda update-function-code --function-name test-ssm-admin-slack-handler --zip-file fileb://bin/admin-slack-handler.zip --region us-east-1
	@echo "Creating new API Gateway deployment..."
	aws apigateway create-deployment --rest-api-id 0c2yacisp9 --stage-name test --region us-east-1
	@echo "Lambda deployment complete!"

# Initialize Terraform
terraform-init:
	cd infrastructure/terraform && terraform init

# Format code
fmt:
	go fmt ./...

# Tidy dependencies
tidy:
	go mod tidy

# Run all checks before commit
pre-commit: fmt tidy lint test
