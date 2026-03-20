# Security group: only ECS tasks can reach MySQL on port 3306
resource "aws_security_group" "rds" {
  name        = "${var.service_name}-rds-sg"
  description = "Allow MySQL access from ECS tasks only"
  vpc_id      = var.vpc_id

  ingress {
    from_port       = 3306
    to_port         = 3306
    protocol        = "tcp"
    security_groups = [var.ecs_security_group_id]
    description     = "MySQL from ECS"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow all outbound"
  }

  tags = {
    Name = "${var.service_name}-rds-sg"
  }
}

# Subnet group: tells RDS which subnets it can be placed in
resource "aws_db_subnet_group" "this" {
  name       = "${lower(var.service_name)}-db-subnet"
  subnet_ids = var.subnet_ids

  tags = {
    Name = "${var.service_name}-db-subnet"
  }
}

# RDS MySQL instance
resource "aws_db_instance" "this" {
  identifier     = "${lower(var.service_name)}-mysql"
  engine         = "mysql"
  engine_version = "8.0"
  instance_class = "db.t3.micro"

  allocated_storage = 20
  storage_type      = "gp2"

  db_name  = var.db_name
  username = var.db_username
  password = var.db_password

  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  # Assignment-friendly settings
  skip_final_snapshot    = true
  deletion_protection    = false
  publicly_accessible    = false

  tags = {
    Name = "${var.service_name}-mysql"
  }
}