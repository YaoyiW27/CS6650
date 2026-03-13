# Terraform Infrastructure

## File Overview

| File | Description |
|------|-------------|
| `main.tf` | AWS provider configuration (us-west-2) |
| `variables.tf` | Input variables: VPC CIDRs, ECR image URIs, worker count |
| `vpc.tf` | VPC, public/private subnets, Internet Gateway, NAT Gateway, route tables |
| `alb.tf` | Application Load Balancer, target group, listener, security groups |
| `sns_sqs.tf` | SNS topic (`order-processing-events`), SQS queue (`order-processing-queue`), SNS→SQS subscription |
| `iam.tf` | References pre-existing `LabRole` for ECS task execution and task roles |
| `ecs.tf` | ECS cluster, task definitions and services for order-receiver and order-processor |
| `outputs.tf` | ALB DNS, SNS topic ARN, SQS queue URL |

## Architecture

```
Internet → ALB (public subnets) → order-receiver (private subnets)
                                        ↓ POST /orders/async
                                   SNS topic
                                        ↓
                                   SQS queue
                                        ↓
                                  order-processor (private subnets)
```

## Prerequisites

- AWS CLI configured with Learner Lab credentials (including session token)
- Docker running
- Terraform installed

## Deployment Steps

### 1. Create ECR repositories and push images

```bash
aws ecr create-repository --repository-name order-receiver --region us-west-2
aws ecr create-repository --repository-name order-processor --region us-west-2

aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin <ACCOUNT_ID>.dkr.ecr.us-west-2.amazonaws.com

ECR_BASE=<ACCOUNT_ID>.dkr.ecr.us-west-2.amazonaws.com

cd order-receiver
docker buildx build --builder singlearch --platform linux/amd64 --push -t $ECR_BASE/order-receiver:latest .
cd ../order-processor
docker buildx build --builder singlearch --platform linux/amd64 --push -t $ECR_BASE/order-processor:latest .
cd ..
```

### 2. Create terraform.tfvars

```hcl
receiver_image  = "<ACCOUNT_ID>.dkr.ecr.us-west-2.amazonaws.com/order-receiver:latest"
processor_image = "<ACCOUNT_ID>.dkr.ecr.us-west-2.amazonaws.com/order-processor:latest"
```

### 3. Deploy

```bash
cd terraform
terraform init
terraform plan
terraform apply
```

### 4. Verify

```bash
curl http://<ALB_DNS>/health
curl -X POST http://<ALB_DNS>/orders/sync -H "Content-Type: application/json" -d '{"customer_id":1,"items":[{"item_id":"A1","name":"Widget","price":9.99,"quantity":2}]}'
curl -X POST http://<ALB_DNS>/orders/async -H "Content-Type: application/json" -d '{"customer_id":2,"items":[{"item_id":"B1","name":"Gadget","price":19.99,"quantity":1}]}'
```

### 5. Scale workers (Phase 5)

Edit `terraform.tfvars`:
```hcl
num_workers = 5  # or 20, 100
```

Then: `terraform apply`

### 6. Cleanup

```bash
terraform destroy
```