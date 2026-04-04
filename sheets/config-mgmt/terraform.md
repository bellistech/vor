# Terraform (Infrastructure as Code)

Declarative IaC tool that provisions and manages cloud resources via providers.

## Core Workflow

### Initialize a project

```bash
terraform init
terraform init -upgrade   # upgrade provider versions
terraform init -backend-config="bucket=my-tfstate"
```

### Plan changes

```bash
terraform plan
terraform plan -out=tfplan            # save plan to file
terraform plan -target=aws_instance.web  # plan a single resource
terraform plan -var="region=us-west-2"
```

### Apply changes

```bash
terraform apply
terraform apply tfplan                # apply a saved plan
terraform apply -auto-approve         # skip confirmation
terraform apply -var-file=prod.tfvars
```

### Destroy infrastructure

```bash
terraform destroy
terraform destroy -target=aws_instance.web
terraform destroy -auto-approve
```

## State Management

### List resources in state

```bash
terraform state list
```

### Show a resource

```bash
terraform state show aws_instance.web
```

### Move a resource (rename)

```bash
terraform state mv aws_instance.web aws_instance.app
```

### Remove from state without destroying

```bash
terraform state rm aws_instance.web
```

### Pull remote state locally

```bash
terraform state pull > terraform.tfstate.backup
```

### Push local state to remote

```bash
terraform state push terraform.tfstate
```

## Import

### Import an existing resource into state

```bash
terraform import aws_instance.web i-0abc123def456
terraform import 'aws_security_group.sg["web"]' sg-0abc123
```

### Generate config for imported resources (1.5+)

```bash
terraform plan -generate-config-out=generated.tf
```

## Output

### Show all outputs

```bash
terraform output
terraform output -json
```

### Show a specific output

```bash
terraform output db_endpoint
terraform output -raw db_password   # no quotes, good for scripting
```

## Workspaces

### List workspaces

```bash
terraform workspace list
```

### Create and switch

```bash
terraform workspace new staging
terraform workspace select production
```

### Use workspace in config

```bash
# resource "aws_instance" "web" {
#   tags = {
#     Environment = terraform.workspace
#   }
# }
```

## Modules

### Use a module

```bash
# module "vpc" {
#   source  = "terraform-aws-modules/vpc/aws"
#   version = "5.1.0"
#   cidr    = "10.0.0.0/16"
# }
```

### Local module

```bash
# module "app" {
#   source = "./modules/app"
#   port   = 8080
# }
```

### Download module sources

```bash
terraform get
terraform get -update
```

## Providers & Data Sources

### Provider config

```bash
# provider "aws" {
#   region  = "us-east-1"
#   profile = "production"
# }
```

### Data source (read existing infra)

```bash
# data "aws_ami" "ubuntu" {
#   most_recent = true
#   owners      = ["099720109477"]
#   filter {
#     name   = "name"
#     values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
#   }
# }
```

### Lock provider versions

```bash
terraform providers lock -platform=linux_amd64 -platform=darwin_amd64
```

## Formatting & Validation

### Format all .tf files

```bash
terraform fmt
terraform fmt -recursive
terraform fmt -check   # exit 1 if unformatted (good for CI)
```

### Validate config

```bash
terraform validate
```

## Taint & Replace

### Mark a resource for recreation (legacy)

```bash
terraform taint aws_instance.web
terraform untaint aws_instance.web
```

### Replace a resource (modern, 0.15.2+)

```bash
terraform apply -replace=aws_instance.web
```

## Console & Graph

### Interactive expression evaluator

```bash
terraform console
# > cidrsubnet("10.0.0.0/16", 8, 1)
# "10.0.1.0/24"
```

### Generate dependency graph

```bash
terraform graph | dot -Tpng > graph.png
```

## Tips

- Always run `terraform plan` before `apply` and review the diff.
- Use `-out=tfplan` to ensure the applied plan matches exactly what you reviewed.
- Never edit `terraform.tfstate` by hand. Use `terraform state` subcommands.
- Remote backends (S3+DynamoDB, Terraform Cloud) prevent state corruption from concurrent runs.
- Pin provider versions in `required_providers` to avoid surprise upgrades.
- Use `terraform fmt -check` in CI to enforce consistent formatting.
- `terraform plan -destroy` previews what `destroy` would remove without actually destroying.
- `count` and `for_each` are not interchangeable. Prefer `for_each` for named resources since `count` uses numeric indexes that shift on removal.
- Store sensitive outputs with `sensitive = true` to hide them from plan output.

## See Also

- ansible
- packer
- aws-cli
- gcloud
- azure-cli
- kubernetes
- vault

## References

- [Terraform Documentation](https://developer.hashicorp.com/terraform/docs)
- [Terraform CLI Reference](https://developer.hashicorp.com/terraform/cli)
- [Terraform Language (HCL) Reference](https://developer.hashicorp.com/terraform/language)
- [Terraform Provider Registry](https://registry.terraform.io/browse/providers)
- [Terraform Module Registry](https://registry.terraform.io/browse/modules)
- [Terraform State Management](https://developer.hashicorp.com/terraform/language/state)
- [Terraform Backend Configuration](https://developer.hashicorp.com/terraform/language/backend)
- [Terraform Built-in Functions](https://developer.hashicorp.com/terraform/language/functions)
- [Terraform Import](https://developer.hashicorp.com/terraform/cli/import)
- [Terraform GitHub Repository](https://github.com/hashicorp/terraform)
- [OpenTofu — Open-source Terraform Fork](https://github.com/opentofu/opentofu)
