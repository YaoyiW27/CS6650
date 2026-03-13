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

data "aws_iam_role" "lab_role" {
  name = "LabRole"
}

# Lambda function
resource "aws_lambda_function" "order_processor" {
  filename         = "../lambda/function.zip"
  function_name    = "order-processor-lambda"
  role             = data.aws_iam_role.lab_role.arn
  handler          = "bootstrap"
  runtime          = "provided.al2"
  memory_size      = 512
  timeout          = 30
  source_code_hash = filebase64sha256("../lambda/function.zip")

  tags = { Name = "hw7-order-processor-lambda" }
}

# Allow SNS to invoke Lambda
resource "aws_lambda_permission" "sns" {
  statement_id  = "AllowSNSInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.order_processor.function_name
  principal     = "sns.amazonaws.com"
  source_arn    = var.sns_topic_arn
}

# Subscribe Lambda to SNS topic
resource "aws_sns_topic_subscription" "lambda" {
  topic_arn = var.sns_topic_arn
  protocol  = "lambda"
  endpoint  = aws_lambda_function.order_processor.arn
}

# CloudWatch log group
resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/order-processor-lambda"
  retention_in_days = 7
}