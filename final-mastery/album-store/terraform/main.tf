provider "aws" {
  region = "us-west-2"
}

data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

# Security group - open 8080 for the app, 5432 for RDS, 22 for SSH
resource "aws_security_group" "album_store" {
  name        = "album-store-sg"
  description = "Album store security group"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port = 5432
    to_port   = 5432
    protocol  = "tcp"
    self      = true
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# S3 bucket
resource "aws_s3_bucket" "photos" {
  bucket_prefix = "album-store-photos-"
  force_destroy = true
}

# SQS queue
resource "aws_sqs_queue" "photos" {
  name                       = "album-store-photos"
  visibility_timeout_seconds = 60
  message_retention_seconds  = 3600
}

# RDS PostgreSQL
resource "aws_db_instance" "postgres" {
  identifier           = "album-store-db"
  engine               = "postgres"
  engine_version       = "16"
  instance_class       = "db.t3.micro"
  allocated_storage    = 20
  db_name              = "albumstore"
  username             = "postgres"
  password             = "postgres123"
  skip_final_snapshot  = true
  publicly_accessible  = true
  vpc_security_group_ids = [aws_security_group.album_store.id]
}

# EC2 instance
resource "aws_instance" "app" {
  ami                    = "ami-05134c8ef96964280" # Ubuntu 24.04 us-west-2
  instance_type          = "t3.medium"
  key_name               = "cs6650-key"
  vpc_security_group_ids = [aws_security_group.album_store.id]

  iam_instance_profile = "LabInstanceProfile"

  user_data = <<-EOF
    #!/bin/bash
    apt-get update
    apt-get install -y golang-go
  EOF

  tags = {
    Name = "album-store"
  }
}