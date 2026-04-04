# Packer (Machine Image Builder)

> Build identical machine images for multiple platforms from a single HCL2 template with builders, provisioners, and post-processors.

## Concepts

### Template Structure

```
# source    — defines the builder (AMI, QEMU, Docker, etc.)
# build     — ties sources to provisioners and post-processors
# variable  — input parameters (var block or .pkrvars.hcl files)
# local     — computed values (locals block)
# data      — external data sources (Amazon AMI lookup, etc.)
# packer    — required_plugins block for plugin management
```

## CLI Commands

### Core Workflow

```bash
# Initialize plugins declared in required_plugins
packer init template.pkr.hcl

# Validate template syntax and config
packer validate template.pkr.hcl
packer validate -var "region=us-east-1" template.pkr.hcl

# Build the image
packer build template.pkr.hcl
packer build -var-file=prod.pkrvars.hcl template.pkr.hcl
packer build -only="amazon-ebs.ubuntu" template.pkr.hcl   # build specific source
packer build -except="docker.dev" template.pkr.hcl         # skip specific source
packer build -parallel-builds=2 template.pkr.hcl           # limit parallelism

# Inspect template (list variables, sources, etc.)
packer inspect template.pkr.hcl

# Format HCL files
packer fmt template.pkr.hcl
packer fmt -recursive .                                     # format all .pkr.hcl files
```

## Variables

### Variable Declaration and Usage

```hcl
# variables.pkr.hcl
variable "region" {
  type        = string
  default     = "us-east-1"
  description = "AWS region for the AMI"
}

variable "instance_type" {
  type    = string
  default = "t3.micro"
}

variable "tags" {
  type = map(string)
  default = {
    Environment = "dev"
    ManagedBy   = "packer"
  }
}

# Local values (computed)
locals {
  timestamp  = formatdate("YYYYMMDD-HHmmss", timestamp())
  ami_name   = "app-${local.timestamp}"
}
```

```bash
# Pass variables at build time
packer build -var "region=eu-west-1" -var "instance_type=t3.small" .

# Use a variable file
# prod.pkrvars.hcl
# region        = "us-west-2"
# instance_type = "t3.medium"
packer build -var-file=prod.pkrvars.hcl .

# Environment variables (PKR_VAR_ prefix)
export PKR_VAR_region="ap-southeast-1"
packer build .
```

## Builders (Sources)

### Amazon EBS

```hcl
packer {
  required_plugins {
    amazon = {
      version = ">= 1.2.0"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

source "amazon-ebs" "ubuntu" {
  ami_name      = local.ami_name
  instance_type = var.instance_type
  region        = var.region

  source_ami_filter {
    filters = {
      name                = "ubuntu/images/*ubuntu-jammy-22.04-amd64-server-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
    }
    most_recent = true
    owners      = ["099720109477"]     # Canonical
  }

  ssh_username = "ubuntu"
  tags         = var.tags
}
```

### QEMU

```hcl
source "qemu" "debian" {
  iso_url          = "https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-12.5.0-amd64-netinst.iso"
  iso_checksum     = "sha256:abcdef..."
  output_directory = "output-debian"
  disk_size        = "20G"
  format           = "qcow2"
  accelerator      = "kvm"
  ssh_username     = "root"
  ssh_password     = "packer"
  ssh_timeout      = "20m"
  shutdown_command  = "shutdown -P now"
  boot_command     = ["<esc><wait>", "auto url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg<enter>"]
  http_directory   = "http"
  headless         = true
}
```

### Docker

```hcl
source "docker" "app" {
  image  = "ubuntu:22.04"
  commit = true
  changes = [
    "ENTRYPOINT [\"/usr/bin/app\"]",
    "EXPOSE 8080",
  ]
}
```

## Provisioners

### Shell Provisioner

```hcl
build {
  sources = ["source.amazon-ebs.ubuntu"]

  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y nginx curl jq",
      "sudo systemctl enable nginx",
    ]
  }

  provisioner "shell" {
    script           = "scripts/setup.sh"
    environment_vars = ["APP_ENV=production", "DB_HOST=10.0.1.5"]
    execute_command  = "chmod +x {{ .Path }}; sudo {{ .Vars }} {{ .Path }}"
  }
}
```

### File Provisioner

```hcl
  provisioner "file" {
    source      = "configs/nginx.conf"
    destination = "/tmp/nginx.conf"
  }

  provisioner "shell" {
    inline = ["sudo mv /tmp/nginx.conf /etc/nginx/nginx.conf"]
  }
```

### Ansible Provisioner

```hcl
  provisioner "ansible" {
    playbook_file = "ansible/playbook.yml"
    extra_arguments = [
      "--extra-vars", "env=production",
      "-v",
    ]
    ansible_env_vars = ["ANSIBLE_HOST_KEY_CHECKING=False"]
  }
```

## Post-Processors

### Manifest

```hcl
  post-processor "manifest" {
    output     = "manifest.json"
    strip_path = true
  }
```

### Vagrant Box

```hcl
  post-processor "vagrant" {
    output               = "builds/{{.Provider}}-{{.BuildName}}.box"
    vagrantfile_template = "templates/Vagrantfile.tpl"
  }
```

### Docker Tag and Push

```hcl
  post-processors {
    post-processor "docker-tag" {
      repository = "registry.example.com/app"
      tags       = ["latest", local.timestamp]
    }
    post-processor "docker-push" {}
  }
```

## Multi-Build Example

### Build for Multiple Platforms

```hcl
build {
  sources = [
    "source.amazon-ebs.ubuntu",
    "source.qemu.debian",
    "source.docker.app",
  ]

  # Runs on all sources
  provisioner "shell" {
    inline = ["echo 'Common setup'"]
  }

  # Only on Amazon
  provisioner "shell" {
    only   = ["amazon-ebs.ubuntu"]
    inline = ["sudo apt-get install -y awscli"]
  }

  # Only on QEMU
  provisioner "shell" {
    only   = ["qemu.debian"]
    inline = ["sudo apt-get install -y qemu-guest-agent"]
  }
}
```

## Tips

- Always run `packer init` before `build` to ensure plugins are installed.
- Use `packer fmt -recursive .` in CI to enforce consistent formatting.
- Set `force_deregister = true` (AWS) to replace existing AMIs with the same name.
- Use `build -on-error=ask` during development to pause on failure for debugging.
- Packer builds are ephemeral; never SSH into a running build instance manually.
- Use data sources (`data "amazon-ami"`) to dynamically find base images rather than hardcoding AMI IDs.
- Chain post-processors in a `post-processors` block (plural) to create a pipeline.

## See Also

- terraform
- ansible
- vagrant
- docker
- cloud-init
- aws-cli

## References

- [Packer Documentation](https://developer.hashicorp.com/packer/docs)
- [Packer HCL2 Template Reference](https://developer.hashicorp.com/packer/docs/templates/hcl_templates)
- [Packer Plugin Registry](https://developer.hashicorp.com/packer/plugins)
- [Packer CLI Reference](https://developer.hashicorp.com/packer/docs/commands)
- [Packer Builder Reference](https://developer.hashicorp.com/packer/docs/builders)
- [Packer Provisioner Reference](https://developer.hashicorp.com/packer/docs/provisioners)
- [Packer Post-Processor Reference](https://developer.hashicorp.com/packer/docs/post-processors)
- [Packer Variables and Locals](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/variables)
- [Packer GitHub Repository](https://github.com/hashicorp/packer)
- [Packer Data Sources](https://developer.hashicorp.com/packer/docs/datasources)
- [Packer Debugging Builds](https://developer.hashicorp.com/packer/docs/debugging)
