variable "service_name" {
  description = "Base name for RDS resources"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID to place RDS in"
  type        = string
}

variable "subnet_ids" {
  description = "Subnet IDs for the DB subnet group"
  type        = list(string)
}

variable "ecs_security_group_id" {
  description = "Security group ID of ECS tasks (allowed to access RDS)"
  type        = string
}

variable "db_name" {
  description = "Name of the database to create"
  type        = string
  default     = "shopping"
}

variable "db_username" {
  description = "Master username for the database"
  type        = string
  default     = "admin"
}

variable "db_password" {
  description = "Master password for the database"
  type        = string
  sensitive   = true
}