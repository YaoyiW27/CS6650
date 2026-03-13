# --- SNS Topic ---
resource "aws_sns_topic" "order_events" {
  name = "order-processing-events"
  tags = { Name = "${var.project_name}-order-events" }
}

# --- SQS Queue ---
resource "aws_sqs_queue" "order_queue" {
  name                       = "order-processing-queue"
  visibility_timeout_seconds = 30
  message_retention_seconds  = 345600 # 4 days
  receive_wait_time_seconds  = 20     # Long polling

  tags = { Name = "${var.project_name}-order-queue" }
}

# --- SQS Queue Policy (allow SNS to send messages) ---
resource "aws_sqs_queue_policy" "order_queue_policy" {
  queue_url = aws_sqs_queue.order_queue.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { Service = "sns.amazonaws.com" }
        Action    = "sqs:SendMessage"
        Resource  = aws_sqs_queue.order_queue.arn
        Condition = {
          ArnEquals = {
            "aws:SourceArn" = aws_sns_topic.order_events.arn
          }
        }
      }
    ]
  })
}

# --- SNS → SQS Subscription ---
resource "aws_sns_topic_subscription" "sqs_subscription" {
  topic_arn = aws_sns_topic.order_events.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.order_queue.arn
}