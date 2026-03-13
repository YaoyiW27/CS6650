# --- ECS Cluster ---
resource "aws_ecs_cluster" "main" {
  name = "${var.project_name}-cluster"
  tags = { Name = "${var.project_name}-cluster" }
}

# --- CloudWatch Log Groups ---
resource "aws_cloudwatch_log_group" "receiver" {
  name              = "/ecs/${var.project_name}-receiver"
  retention_in_days = 7
}

resource "aws_cloudwatch_log_group" "processor" {
  name              = "/ecs/${var.project_name}-processor"
  retention_in_days = 7
}

# =====================
# Order Receiver
# =====================
resource "aws_ecs_task_definition" "receiver" {
  family                   = "${var.project_name}-receiver"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512
  execution_role_arn       = data.aws_iam_role.lab_role.arn
  task_role_arn            = data.aws_iam_role.lab_role.arn

  container_definitions = jsonencode([
    {
      name      = "receiver"
      image     = var.receiver_image
      essential = true
      portMappings = [
        { containerPort = 8080, protocol = "tcp" }
      ]
      environment = [
        { name = "SNS_TOPIC_ARN", value = aws_sns_topic.order_events.arn },
        { name = "AWS_REGION", value = var.aws_region },
        { name = "PORT", value = "8080" }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.receiver.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "receiver"
        }
      }
    }
  ])
}

resource "aws_ecs_service" "receiver" {
  name            = "${var.project_name}-receiver"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.receiver.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = aws_subnet.private[*].id
    security_groups  = [aws_security_group.ecs.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.receiver.arn
    container_name   = "receiver"
    container_port   = 8080
  }

  depends_on = [aws_lb_listener.http]
}

# =====================
# Order Processor
# =====================
resource "aws_ecs_task_definition" "processor" {
  family                   = "${var.project_name}-processor"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512
  execution_role_arn       = data.aws_iam_role.lab_role.arn
  task_role_arn            = data.aws_iam_role.lab_role.arn

  container_definitions = jsonencode([
    {
      name      = "processor"
      image     = var.processor_image
      essential = true
      portMappings = [
        { containerPort = 8080, protocol = "tcp" }
      ]
      environment = [
        { name = "SQS_QUEUE_URL", value = aws_sqs_queue.order_queue.url },
        { name = "AWS_REGION", value = var.aws_region },
        { name = "NUM_WORKERS", value = tostring(var.num_workers) }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.processor.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "processor"
        }
      }
    }
  ])
}

resource "aws_ecs_service" "processor" {
  name            = "${var.project_name}-processor"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.processor.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = aws_subnet.private[*].id
    security_groups  = [aws_security_group.ecs.id]
    assign_public_ip = false
  }
}