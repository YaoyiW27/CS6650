output "alb_dns" {
  description = "ALB DNS name for testing"
  value       = aws_lb.main.dns_name
}

output "sns_topic_arn" {
  description = "SNS topic ARN"
  value       = aws_sns_topic.order_events.arn
}

output "sqs_queue_url" {
  description = "SQS queue URL"
  value       = aws_sqs_queue.order_queue.url
}