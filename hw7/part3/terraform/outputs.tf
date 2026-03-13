output "lambda_function_name" {
  value = aws_lambda_function.order_processor.function_name
}

output "lambda_arn" {
  value = aws_lambda_function.order_processor.arn
}