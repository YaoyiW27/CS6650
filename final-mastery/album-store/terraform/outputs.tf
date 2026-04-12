output "ec2_public_ip" {
  value = aws_instance.app.public_ip
}

output "rds_endpoint" {
  value = aws_db_instance.postgres.endpoint
}

output "s3_bucket" {
  value = aws_s3_bucket.photos.bucket
}

output "sqs_queue_url" {
  value = aws_sqs_queue.photos.url
}