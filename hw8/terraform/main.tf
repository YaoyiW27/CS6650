terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 2.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# Docker provider for building & pushing images to ECR
data "aws_ecr_authorization_token" "registry" {}

provider "docker" {
  registry_auth {
    address  = data.aws_ecr_authorization_token.registry.proxy_endpoint
    username = data.aws_ecr_authorization_token.registry.user_name
    password = data.aws_ecr_authorization_token.registry.password
  }
}

# ─── Network: reuse default VPC ───────────────────────────────────────
data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

# ECS security group
resource "aws_security_group" "ecs" {
  name        = "${var.service_name}-ecs-sg"
  description = "Allow inbound on ${var.container_port}"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    from_port   = var.container_port
    to_port     = var.container_port
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow HTTP traffic"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow all outbound"
  }
}

# Reuse existing IAM role
data "aws_iam_role" "lab_role" {
  name = "LabRole"
}

# ─── ECR ──────────────────────────────────────────────────────────────
module "ecr" {
  source          = "./modules/ecr"
  repository_name = var.ecr_repository_name
}

# ─── Logging ──────────────────────────────────────────────────────────
module "logging" {
  source            = "./modules/logging"
  service_name      = var.service_name
  retention_in_days = 7
}

# ─── RDS MySQL (STEP I) ──────────────────────────────────────────────
module "rds" {
  source = "./modules/rds"

  service_name          = var.service_name
  vpc_id                = data.aws_vpc.default.id
  subnet_ids            = data.aws_subnets.default.ids
  ecs_security_group_id = aws_security_group.ecs.id
  db_password           = var.db_password
}

# ─── DynamoDB (STEP II) ───────────────────────────────────────────────
module "dynamodb" {
  source       = "./modules/dynamodb"
  service_name = var.service_name
}

# ─── ECS ──────────────────────────────────────────────────────────────
module "ecs" {
  source = "./modules/ecs"

  service_name       = var.service_name
  image              = "${module.ecr.repository_url}:latest"
  container_port     = var.container_port
  subnet_ids         = data.aws_subnets.default.ids
  security_group_ids = [aws_security_group.ecs.id]
  execution_role_arn = data.aws_iam_role.lab_role.arn
  task_role_arn      = data.aws_iam_role.lab_role.arn
  log_group_name     = module.logging.log_group_name
  ecs_count          = 1
  region             = var.aws_region

  environment = [
    { name = "STORE_TYPE",  value = "dynamodb" },
    { name = "DB_HOST",     value = module.rds.host },
    { name = "DB_PORT",     value = "3306" },
    { name = "DB_USER",     value = "admin" },
    { name = "DB_PASSWORD", value = var.db_password },
    { name = "DB_NAME",     value = "shopping" },
    { name = "PORT",        value = tostring(var.container_port) },
    { name = "DYNAMO_TABLE", value = module.dynamodb.table_name },
    { name = "AWS_REGION",   value = var.aws_region },
  ]
}

# ─── Docker Build & Push ─────────────────────────────────────────────
resource "docker_image" "app" {
  name = "${module.ecr.repository_url}:latest"
  build {
    context = "../"
  }
}

resource "docker_registry_image" "app" {
  name = docker_image.app.name
}

# ─── Outputs ─────────────────────────────────────────────────────────
output "rds_endpoint" {
  description = "MySQL RDS endpoint"
  value       = module.rds.endpoint
}

output "rds_host" {
  description = "MySQL RDS hostname"
  value       = module.rds.host
}

output "ecr_url" {
  description = "ECR repository URL"
  value       = module.ecr.repository_url
}