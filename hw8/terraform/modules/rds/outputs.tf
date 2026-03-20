output "endpoint" {
  description = "RDS endpoint (host:port)"
  value       = aws_db_instance.this.endpoint
}

output "host" {
  description = "RDS hostname only"
  value       = aws_db_instance.this.address
}

output "port" {
  description = "RDS port"
  value       = aws_db_instance.this.port
}

output "db_name" {
  description = "Database name"
  value       = aws_db_instance.this.db_name
}

output "security_group_id" {
  description = "Security group ID of the RDS instance"
  value       = aws_security_group.rds.id
}