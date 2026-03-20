variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "service_name" {
  type    = string
  default = "hw8-cart"
}

variable "container_port" {
  type    = number
  default = 8080
}

variable "db_password" {
  description = "Master password for RDS MySQL"
  type        = string
  sensitive   = true
}