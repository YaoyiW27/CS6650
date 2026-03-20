terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
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

# ECS security group (same pattern as HW5)
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

# Reuse existing IAM role for ECS tasks
data "aws_iam_role" "lab_role" {
  name = "LabRole"
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

# ─── Outputs ─────────────────────────────────────────────────────────
output "rds_endpoint" {
  description = "MySQL RDS endpoint"
  value       = module.rds.endpoint
}

output "rds_host" {
  description = "MySQL RDS hostname"
  value       = module.rds.host
}