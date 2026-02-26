terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# DynamoDB Tables will be defined here
# Lambda functions will be defined here
# API Gateway will be defined here
# EventBridge rules will be defined here
