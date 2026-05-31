.PHONY: test build clean deploy lint deploy-lambdas terraform-init fmt tidy pre-commit test-integration

TF_DIR := infrastructure/terraform

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
	cd $(TF_DIR) && terraform apply

# Deploy Lambda functions only (after build)
# Deployment targets are read from Terraform outputs so no environment-specific
# identifiers (function names, API Gateway ID, region) are hardcoded here.
deploy-lambdas: build
	@echo "Reading deployment targets from Terraform outputs..."
	@region=$$(terraform -chdir=$(TF_DIR) output -raw aws_region) && \
	env=$$(terraform -chdir=$(TF_DIR) output -raw environment) && \
	api_id=$$(terraform -chdir=$(TF_DIR) output -raw rest_api_id) && \
	stage=$$(terraform -chdir=$(TF_DIR) output -raw api_gateway_stage) && \
	for name in request-handler approval-handler document-creator expiration-cleanup admin-handler admin-slack-handler; do \
		fn="$$env-ssm-$$name"; \
		echo "Updating $$fn..."; \
		aws lambda update-function-code --function-name "$$fn" --zip-file "fileb://bin/$$name.zip" --region "$$region" >/dev/null; \
	done && \
	echo "Creating API Gateway deployment ($$api_id, stage $$stage)..." && \
	aws apigateway create-deployment --rest-api-id "$$api_id" --stage-name "$$stage" --region "$$region" >/dev/null && \
	echo "Lambda deployment complete!"

# Initialize Terraform
terraform-init:
	cd $(TF_DIR) && terraform init

# Format code
fmt:
	gofmt -s -w .

# Tidy dependencies
tidy:
	go mod tidy

# Run all checks before commit
pre-commit: fmt tidy lint test
