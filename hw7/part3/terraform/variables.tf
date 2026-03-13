variable "aws_region" {
  type    = string
  default = "us-west-2"
}

variable "sns_topic_arn" {
  description = "SNS topic ARN from Part II"
  type        = string
}