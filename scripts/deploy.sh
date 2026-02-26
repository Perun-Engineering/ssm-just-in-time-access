#!/bin/bash
set -e

# Deployment script for SSM Access Manager

ENVIRONMENT=${1:-dev}
AWS_REGION=${2:-us-east-1}

echo "Deploying SSM Access Manager to environment: $ENVIRONMENT in region: $AWS_REGION"

# Build Lambda functions
echo "Step 1: Building Lambda functions..."
./scripts/build.sh

# Initialize Terraform
echo "Step 2: Initializing Terraform..."
cd infrastructure/terraform
terraform init

# Plan Terraform changes
echo "Step 3: Planning Terraform changes..."
terraform plan \
  -var="environment=$ENVIRONMENT" \
  -var="aws_region=$AWS_REGION" \
  -out=tfplan

# Apply Terraform changes
echo "Step 4: Applying Terraform changes..."
read -p "Do you want to apply these changes? (yes/no): " confirm
if [ "$confirm" = "yes" ]; then
  terraform apply tfplan
  rm tfplan
  
  echo ""
  echo "Deployment complete!"
  echo ""
  echo "API Endpoints:"
  terraform output -raw slack_command_endpoint
  echo ""
  terraform output -raw slack_interaction_endpoint
  echo ""
  terraform output -raw admin_endpoint
  echo ""
else
  echo "Deployment cancelled"
  rm tfplan
  exit 1
fi

cd ../..
