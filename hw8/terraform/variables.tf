variable "aws_region" {
  type    = string
  default = "us-west-2"
}

variable "service_name" {
  type    = string
  default = "hw8-cart"
}

variable "container_port" {
  type    = number
  default = 8080
}

variable "ecr_repository_name" {
  type    = string
  default = "hw8-cart-repo"
}

variable "db_password" {
  description = "Master password for RDS MySQL"
  type        = string
  sensitive   = true
}