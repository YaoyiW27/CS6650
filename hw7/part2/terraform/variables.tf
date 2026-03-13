variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-west-2"
}

variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
  default     = "hw7"
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

variable "public_subnets" {
  description = "Public subnet CIDRs (for ALB)"
  type        = list(string)
  default     = ["10.0.1.0/24", "10.0.2.0/24"]
}

variable "private_subnets" {
  description = "Private subnet CIDRs (for ECS)"
  type        = list(string)
  default     = ["10.0.10.0/24", "10.0.11.0/24"]
}

variable "receiver_image" {
  description = "ECR image URI for order-receiver"
  type        = string
}

variable "processor_image" {
  description = "ECR image URI for order-processor"
  type        = string
}

variable "num_workers" {
  description = "Number of worker goroutines in order-processor"
  type        = number
  default     = 1
}