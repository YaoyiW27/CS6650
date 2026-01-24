# Homework 2 - MapReduce, Terraform, Docker... and Claude Code

## Part II: Terraform EC2 Infrastructure

This part demonstrates Infrastructure as Code (IaC) using Terraform to automate EC2 deployment on AWS.

### Overview

Instead of manually clicking through AWS Console, we use Terraform to define and deploy cloud infrastructure with code. This enables reproducibility and version control of infrastructure.

### Architecture

```
terraform.tfvars (secrets - gitignored)
      │
      ├──▶ ssh_cidr ──────────────────┐
      │                               │
      └──▶ ssh_key_name ──┐           │
                          │           ▼
┌─────────────────┐       │    ┌──────────────────────┐
│  data.aws_ami   │       │    │  aws_security_group  │
│  (AL2023 AMI)   │       │    │  (SSH from your IP)  │
└────────┬────────┘       │    └───────────┬──────────┘
         │                │                │
         │    ┌───────────┘                │
         │    │                            │
         ▼    ▼                            ▼
      ┌─────────────────────────────────────┐
      │          aws_instance               │
      │     (t2.micro EC2 instance)         │
      └──────────────────┬──────────────────┘
                         │
                         ▼
                  ┌─────────────┐
                  │   output    │
                  │ (public DNS)│
                  └─────────────┘
```

### Files

| File | Description |
|------|-------------|
| `main.tf` | Terraform configuration defining EC2, security group, and AMI lookup |
| `terraform.tfvars` | Variables (IP address, key pair name) - **not committed** |
| `.terraform.lock.hcl` | Provider version lock file (committed) |
| `.gitignore` | Excludes sensitive files from version control |

### Resources Created

| Resource | Purpose |
|----------|---------|
| `aws_security_group.ssh` | Allows SSH (port 22) only from my IP |
| `aws_instance.demo-instance` | t2.micro EC2 instance running Amazon Linux 2023 |

### Usage

#### Prerequisites
- Terraform installed
- AWS CLI installed
- AWS Academy Lab credentials
- Existing AWS key pair

#### Setup & Deploy

```
# 1. Configure AWS credentials
aws configure
aws configure set aws_session_token <YOUR_SESSION_TOKEN>

# 2. Create terraform.tfvars with your values
ssh_cidr     = "YOUR_PUBLIC_IP/32"
ssh_key_name = "your-key-name"

# 3. Initialize and deploy
terraform init
terraform apply -auto-approve

# 4. SSH into instance
ssh -i <PATH-TO-KEY.pem> ec2-user@<EC2-PUBLIC-DNS>

# 5. Clean up when done
terraform destroy -auto-approve
```

### Troubleshooting on macOS (Apple Silicon: M1/M2/M3)

#### Problem

On Apple Silicon Macs, Terraform may fail with:

```
Failed to load plugin schemas
timeout while waiting for plugin to start
```

or Terraform may hang during apply/destroy.

#### Root Cause

This happens when:
- Terraform is installed as **x86_64 (darwin_amd64)** using Intel Homebrew (`/usr/local`)
- But the machine is **arm64**
- Result: AWS provider is also downloaded as `darwin_amd64`, which runs unstably under Rosetta and causes plugin startup timeouts, high CPU usage, and hanging apply/destroy

#### Correct Fix (Permanent)

Use **native arm64 Terraform** from Apple Silicon Homebrew:

```
# Ensure arm64 Homebrew is used
eval "$(/opt/homebrew/bin/brew shellenv)"

# Install arm64 Terraform
brew install terraform

# Verify architecture
terraform version
# Must show: darwin_arm64
```

Then in the project:

```
rm -rf .terraform .terraform.lock.hcl
terraform init
```

Verify provider architecture:

```
file .terraform/providers/.../terraform-provider-aws_*
# Must show: arm64
```

After this, `terraform apply` and `terraform destroy` work reliably.

#### Quick Fix (if above doesn't work)

Remove macOS quarantine attribute:

```bash
xattr -dr com.apple.quarantine .terraform
xattr -dr com.apple.quarantine ~/.terraform.d 2>/dev/null || true
rm -rf .terraform .terraform.lock.hcl
terraform init
terraform apply -auto-approve
```

### Notes

- The EC2 instance will receive a new public IP/DNS each time it is recreated
- AWS Academy credentials expire periodically and must be reconfigured
- Update `ssh_cidr` if your IP address changes (e.g., switching networks)