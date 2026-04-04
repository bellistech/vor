# terragrunt (Terraform wrapper)

Terragrunt is a thin wrapper for Terraform/OpenTofu that provides DRY configuration through dependency management, remote state generation, multi-module orchestration with run_all, and scaffolding for consistent infrastructure code across environments and accounts.

## Project Structure

### Typical Layout

```bash
# Recommended directory structure
# infra-live/
#   terragrunt.hcl          (root config)
#   dev/
#     env.hcl
#     vpc/
#       terragrunt.hcl
#     rds/
#       terragrunt.hcl
#   staging/
#     env.hcl
#     vpc/
#       terragrunt.hcl
#   prod/
#     env.hcl
#     vpc/
#       terragrunt.hcl

# Install Terragrunt
brew install terragrunt                    # macOS
curl -sL https://github.com/gruntwork-io/terragrunt/releases/download/v0.67.0/terragrunt_linux_amd64 -o terragrunt
chmod +x terragrunt && sudo mv terragrunt /usr/local/bin/

# Check version
terragrunt --version
```

## Root Configuration

### Root terragrunt.hcl

```hcl
# Root terragrunt.hcl -- inherited by all child modules

remote_state {
  backend = "s3"
  generate = {
    path      = "backend.tf"
    if_exists = "overwrite_terragrunt"
  }
  config = {
    bucket         = "mycompany-terraform-state"
    key            = "${path_relative_to_include()}/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "terraform-locks"
  }
}

generate "provider" {
  path      = "provider.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<EOF
provider "aws" {
  region = var.aws_region
  default_tags {
    tags = {
      ManagedBy   = "Terragrunt"
      Environment = var.environment
    }
  }
}
EOF
}
```

## Include Blocks

### Inherit Parent Configuration

```hcl
# Child module terragrunt.hcl (e.g., dev/vpc/terragrunt.hcl)

include "root" {
  path = find_in_parent_folders()
}

include "env" {
  path   = "${dirname(find_in_parent_folders())}/common/vpc.hcl"
  expose = true
}

# Access exposed values from include
inputs = {
  vpc_cidr = include.env.locals.vpc_cidr
}
```

### Locals

```hcl
# Environment-level locals (e.g., dev/env.hcl)
locals {
  environment = "dev"
  aws_region  = "us-east-1"
  account_id  = "123456789012"
}

# Read locals from parent
locals {
  env_vars     = read_terragrunt_config(find_in_parent_folders("env.hcl"))
  environment  = local.env_vars.locals.environment
  aws_region   = local.env_vars.locals.aws_region
}
```

## Dependency Management

### Cross-Module Dependencies

```hcl
# rds/terragrunt.hcl -- depends on vpc module
dependency "vpc" {
  config_path = "../vpc"

  # Mock outputs for plan when dependency hasn't been applied
  mock_outputs = {
    vpc_id            = "vpc-mock"
    private_subnet_ids = ["subnet-mock-1", "subnet-mock-2"]
  }
  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
}

dependency "sg" {
  config_path = "../security-groups"
}

inputs = {
  vpc_id     = dependency.vpc.outputs.vpc_id
  subnet_ids = dependency.vpc.outputs.private_subnet_ids
  sg_id      = dependency.sg.outputs.db_security_group_id
}
```

### Dependencies Block (Order Only)

```hcl
# Ensure modules run in order without reading outputs
dependencies {
  paths = ["../vpc", "../iam"]
}
```

## Inputs

### Passing Variables to Terraform

```hcl
# Merge inputs from multiple sources
inputs = merge(
  local.env_vars.locals,
  local.region_vars.locals,
  {
    instance_type = "t3.medium"
    db_name       = "myapp_${local.environment}"
    tags = {
      Project     = "myapp"
      Environment = local.environment
    }
  }
)
```

## Generate Blocks

### Auto-Generate Terraform Files

```hcl
# Generate backend configuration
generate "backend" {
  path      = "backend.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<EOF
terraform {
  backend "s3" {}
}
EOF
}

# Generate versions file
generate "versions" {
  path      = "versions.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<EOF
terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
EOF
}
```

## Terraform Source

### Module Source Configuration

```hcl
# Reference a Terraform module
terraform {
  source = "git::git@github.com:myorg/modules.git//vpc?ref=v1.2.0"
}

# Local module reference
terraform {
  source = "${dirname(find_in_parent_folders())}/modules//vpc"
}

# Terraform Registry module
terraform {
  source = "tfr:///hashicorp/consul/aws?version=0.1.0"
}

# Extra arguments for terraform commands
terraform {
  extra_arguments "retry_lock" {
    commands = get_terraform_commands_that_need_locking()
    arguments = ["-lock-timeout=10m"]
  }

  extra_arguments "plan_file" {
    commands = ["plan"]
    arguments = ["-out=tfplan"]
  }
}
```

## run_all Commands

### Multi-Module Orchestration

```bash
# Apply all modules (respects dependency order)
terragrunt run-all apply

# Plan all modules
terragrunt run-all plan

# Destroy all modules (reverse dependency order)
terragrunt run-all destroy

# Apply with auto-approve
terragrunt run-all apply --terragrunt-non-interactive

# Apply specific environment
cd dev && terragrunt run-all apply

# Limit parallelism
terragrunt run-all apply --terragrunt-parallelism 3

# Exclude specific modules
terragrunt run-all apply --terragrunt-exclude-dir "*/monitoring"

# Include specific modules only
terragrunt run-all apply --terragrunt-include-dir "*/vpc" --terragrunt-include-dir "*/rds"

# Show dependency graph
terragrunt graph-dependencies | dot -Tpng > deps.png

# Output from all modules
terragrunt run-all output --terragrunt-non-interactive
```

## Hooks

### Before and After Hooks

```hcl
terraform {
  before_hook "validate" {
    commands = ["apply", "plan"]
    execute  = ["tflint", "--init"]
  }

  before_hook "checkov" {
    commands = ["apply"]
    execute  = ["checkov", "-d", "."]
  }

  after_hook "notify" {
    commands     = ["apply"]
    execute      = ["bash", "-c", "echo 'Applied ${path_relative_to_include()}'"]
    run_on_error = false
  }

}
```

## Helper Functions

### Built-in Functions

```hcl
locals {
  # Find parent config
  root_config = find_in_parent_folders()

  # Path helpers
  relative_path = path_relative_to_include()

  # Read other configs
  account_vars = read_terragrunt_config(find_in_parent_folders("account.hcl"))
  region_vars  = read_terragrunt_config(find_in_parent_folders("region.hcl"))

  # Scaffold new modules
  # terragrunt scaffold github.com/myorg/modules//vpc
  # terragrunt catalog
}
```

## Tips

- Use `find_in_parent_folders()` to locate the root config and avoid hardcoded paths
- Always set `mock_outputs` on dependencies so `plan` and `validate` work before dependencies are applied
- Use `run-all` with `--terragrunt-parallelism` to control blast radius in CI/CD pipelines
- The `path_relative_to_include()` function generates unique state keys per module automatically
- Use `read_terragrunt_config()` to compose configs from account, region, and environment layers
- Generate blocks keep boilerplate (backend, provider, versions) DRY across hundreds of modules
- Use `error_hook` to send alerts on failed applies without wrapping terragrunt in shell scripts
- Pin module sources to git tags (`?ref=v1.2.0`) for reproducible deployments
- Run `terragrunt graph-dependencies` to visualize and verify the dependency tree before applying
- Set `--terragrunt-exclude-dir` to skip modules during targeted deployments or migrations

## See Also

- terraform, pulumi, crossplane, ansible, make

## References

- [Terragrunt Documentation](https://terragrunt.gruntwork.io/docs/)
- [Terragrunt GitHub](https://github.com/gruntwork-io/terragrunt)
- [Terragrunt Quick Start](https://terragrunt.gruntwork.io/docs/getting-started/quick-start/)
- [Keep Your Terraform Code DRY](https://terragrunt.gruntwork.io/docs/features/keep-your-terraform-code-dry/)
- [Gruntwork Reference Architecture](https://gruntwork.io/reference-architecture/)
