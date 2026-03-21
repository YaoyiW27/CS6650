# DynamoDB table for shopping carts
# Single-table design: cart + items stored together
# Partition key: cart_id (UUID for even distribution)
resource "aws_dynamodb_table" "shopping_carts" {
  name         = "${var.service_name}-carts"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "cart_id"

  attribute {
    name = "cart_id"
    type = "S"
  }

  tags = {
    Name = "${var.service_name}-carts"
  }
}