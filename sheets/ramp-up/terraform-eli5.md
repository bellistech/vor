# Terraform — ELI5 (Blueprints for Cloud Infrastructure)

> Terraform is a robot that reads your blueprint and builds your cloud for you, then remembers exactly what it built so you can change it or tear it down later without losing your mind.

## Prerequisites

(none — but a few things help)

You do not need to know what Terraform is to read this sheet. You do not need to have ever written a `.tf` file. You do not need to know what HCL means. By the end of this sheet you will know all of that, and you will have run real `terraform` commands and watched real cloud resources appear and disappear.

A few small things will make this sheet easier:

- **Knowing what an EC2 instance, a GCP VM, or an Azure VM is** — basically, "a server in somebody else's data center that you rent by the hour." If you have ever clicked "Launch instance" in the AWS Console, you already get the idea. If you haven't, that's fine too — pretend it is a virtual computer in the cloud and that's good enough for this sheet.
- **Reading `cs ramp-up linux-kernel-eli5`** if you want to know what is actually running on those VMs. Terraform creates the VM. The Linux kernel runs on it. Two different jobs.
- **A vague sense of what "the cloud" means** — it means renting servers, databases, networks, and other computing things from a giant company (Amazon, Google, Microsoft, etc.) instead of buying physical hardware and putting it in a closet in your office. You pay by the hour, by the gigabyte, by the request.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is Terraform?

### The robot and the blueprint

Imagine you want to build a treehouse. You could grab a hammer, climb the tree, and start nailing boards. That is one way. It is messy. You will probably forget what you nailed where. If you want to add a window later you will have to climb back up and remember what the wall looks like. If you want to tear it down later you will have to remember every nail.

Or, you could write down on a piece of paper exactly what you want: "I want a treehouse with a square floor, four walls, one window on the east side, one door on the south side, and a roof." Then you give that piece of paper to a robot. The robot reads the paper, climbs the tree, and builds exactly what you asked for. The robot also writes down what it built, and where each board is, on a little notebook.

Now if you want to add a window, you do not climb the tree. You change the piece of paper. You add a line that says "and one window on the west side too." You give the paper back to the robot. The robot reads the paper, looks at its notebook, sees that the west wall does not have a window yet, and adds one. It does not rebuild the whole treehouse. It does not knock down the east window. It only does the new thing.

If you want to tear the whole treehouse down, you do not climb the tree with a crowbar. You tell the robot "tear it all down." The robot reads its notebook, finds every board it nailed, and removes them in the correct order. When the robot is done there is no treehouse and the notebook is empty.

**That robot is Terraform.**

The piece of paper is your `.tf` file. The notebook is the **state file**. The treehouse is your cloud infrastructure: your servers, your databases, your load balancers, your DNS records, your firewall rules, all of it.

### What Terraform actually does

Terraform is a command-line tool. You write text files (called `.tf` files) describing what you want your cloud to look like. You run `terraform apply` and Terraform calls the cloud's APIs (the same APIs the cloud's web console uses behind the scenes) to make your cloud look like what you described.

Terraform does three things really, really well:

1. **It reads your description and figures out what to do.** You write "I want 3 EC2 instances and 1 RDS database." Terraform figures out the right order, the right API calls, the right parameters.
2. **It calls the cloud APIs for you.** You do not have to know the difference between `aws ec2 run-instances` and `aws ec2 modify-instance-attribute`. Terraform handles the calls.
3. **It remembers what it did.** This is the magic part. Terraform writes down every resource it created — the EC2 instance ID, the RDS endpoint, the security group ID, all of it — into a file called the **state file**. The next time you run Terraform, it reads the state file first so it knows what already exists.

This is the core trick. Other tools just call APIs. Terraform calls APIs **and** remembers what it did, which means the next time you run it, it can compare what's currently real to what your blueprint says, and apply just the difference.

### Declarative, not imperative

There are two ways to tell a computer what to do.

**Imperative** is when you give step-by-step instructions: "Open the door. Walk to the kitchen. Take a plate. Put it on the table. Open the fridge. Take out the cheese." You are listing actions. The order matters. If you skip a step the whole thing breaks.

**Declarative** is when you describe the end result: "I want a plate of cheese on the table." Somebody else figures out the steps. If the table already has a plate, they just put cheese on it. If the cheese is already out, they grab it. If the plate is dirty, they wash it first. You don't care how. You just care that, when they're done, there is a plate of cheese on the table.

Terraform is declarative. You describe the end state. Terraform figures out how to get there.

This is why a `.tf` file looks like a description, not a script. There are no `if` statements wrapping every API call. You don't write "if the security group does not exist, create it." You write "this security group should exist." Terraform figures out the if for you.

### What if I have nothing yet?

Then Terraform creates everything. Your blueprint says "3 EC2 instances and 1 RDS database." Your cloud account is empty. Terraform calls the right APIs, creates 3 EC2 instances and 1 RDS database, writes them all down in the state file. Done.

### What if I already have stuff that matches the blueprint?

Then Terraform does nothing. Your blueprint says "3 EC2 instances." You already have 3 EC2 instances that match (Terraform knows because the state file says so). Terraform reads the blueprint, reads the state file, sees they match, and exits with "No changes."

### What if I change the blueprint?

This is the everyday case. You had 3 instances, now you change the blueprint to say 5. You run `terraform apply`. Terraform reads the blueprint, reads the state file, sees that 3 already exist and 2 are missing, and creates only the 2 missing ones. The existing 3 are left alone.

### What if I delete a line from the blueprint?

Terraform sees the blueprint says nothing about that resource anymore, but the state file says it does exist. Terraform interprets that as "you want this gone." It calls the destroy API for that resource. Then it removes the resource from the state file.

### What if I change a setting on a resource?

Same idea. The blueprint now says `instance_type = "t3.large"` instead of `t3.micro`. Terraform reads the blueprint, reads the state file, sees the difference, and figures out the right API call. Some changes can be done in place (the AWS API supports modify-instance-attribute), some require destroying and recreating the resource. Terraform shows you which it'll do BEFORE doing it (this is the **plan**).

### Multi-cloud, multi-vendor

Terraform isn't just for AWS. It works with hundreds of different services. Each one has a thing called a **provider** — a small plugin that knows how to talk to that service's API. There are providers for:

- **AWS, GCP, Azure** — the big three clouds.
- **Cloudflare, Fastly, Akamai** — CDN and DNS providers.
- **GitHub, GitLab** — Terraform can manage your repos, your branch protection rules, your deploy keys.
- **Kubernetes, Helm** — Terraform can manage k8s clusters and deployments.
- **PagerDuty, Datadog, New Relic** — Terraform can manage your alerts and dashboards.
- **Vault, Consul, Nomad** — HashiCorp's other tools.
- **Stripe, Snowflake, MongoDB Atlas** — even SaaS products you'd think aren't infrastructure-y.
- **Local files** — yes, even your local file system, for templating.

This means you can describe an entire production stack in one set of `.tf` files: AWS for the servers, Cloudflare for the DNS, PagerDuty for the alerts, GitHub for the repo permissions. One blueprint. One robot. Many places.

### HCL: HashiCorp Configuration Language

The text format of `.tf` files is called **HCL**, which stands for **HashiCorp Configuration Language** (HashiCorp is the company that made Terraform). HCL looks a bit like JSON or YAML, but it is its own thing. It has blocks, attributes, expressions, and a small embedded scripting language for calculating values.

A tiny HCL example:

```hcl
resource "aws_instance" "web" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.micro"
}
```

That is one block. The block type is `resource`. The block has two labels: `"aws_instance"` and `"web"`. Inside the block are two attributes: `ami` and `instance_type`. Each attribute has a value.

This is the entire shape of HCL. Blocks with labels. Attributes inside. That's it. You'll get used to it within an hour.

### The state file is the brain

We will talk about state in much more detail later, but it is so important it deserves a quick mention up front.

The state file (`terraform.tfstate`) is a JSON file that maps your blueprint to your real cloud resources. Without the state file, Terraform has no idea what already exists. It would either skip everything (because it thinks the cloud is empty) or try to create duplicates of everything (because it doesn't know it already created them).

The state file is precious. **Never edit it by hand.** **Never check it into git.** **Always store it remotely** (in S3, GCS, Azure Blob, or Terraform Cloud) so the whole team can share it. **Always lock it** so two people don't apply at the same time and corrupt it.

If you take nothing else from this sheet, take this: **the state file is the brain. Protect the brain.**

### Why people love Terraform

People love Terraform because it solves a real, painful problem: cloud infrastructure used to live in people's heads, in screenshots in Confluence, in a tribal-knowledge document that hadn't been updated since 2018. Nobody really knew what was in production. Nobody dared change anything because they didn't know what would break.

With Terraform, the production cloud is described in version-controlled text. You can read it. You can review it in a pull request. You can roll back to a previous git commit. You can spin up an identical staging environment in 5 minutes by copying the same code and changing one variable.

It also does the boring parts for you. You don't have to remember the exact order to delete a VPC and all its dependent resources (you have to delete the route table associations first, then the routes, then the gateway attachments, then the subnets, then the route tables, then the gateways, then finally the VPC — Terraform knows). You just say "delete this VPC" and Terraform figures out the order.

### The 30-second pitch

If you take a friend out for coffee and they ask "what's Terraform?", here is what you say:

> Terraform is a tool where you write a text file describing what you want your cloud to look like — servers, databases, network rules, all of it — and it calls the cloud's APIs to make it real, AND remembers everything it created so you can change or tear down later just by editing the text file. It works for any cloud. It's free. It's the standard.

That is the whole pitch. Everything else is detail.

## The Three Steps

Terraform has a lot of commands, but day-to-day you mostly use three:

```
terraform init    # download providers, set up state backend
terraform plan    # show what would change (no changes made)
terraform apply   # actually do it
```

And one more for tearing down:

```
terraform destroy # tear it all down
```

Let's walk through each one.

### terraform init

`init` is the very first thing you ever run in a new Terraform project. Or after you change which providers you're using. Or after somebody else changes the backend configuration.

What `init` does:

1. **Reads your `.tf` files** to find out which providers you need (AWS? GCP? Cloudflare?).
2. **Downloads those providers** from the Terraform Registry. They get cached in a hidden folder called `.terraform/`.
3. **Sets up the state backend** — wherever Terraform should store the state file (local disk, S3, GCS, Terraform Cloud, etc.).
4. **Creates a lockfile** called `.terraform.lock.hcl` that pins exact provider versions, so your teammates get the same versions when they run `init`.

Picture:

```
Your .tf files                        Internet
+----------------+                +-------------------+
| terraform {    |                | Terraform Registry|
|   required_    |    init -->    | (registry.        |
|   providers {} |                |  terraform.io)    |
| }              |                |                   |
+----------------+                |  [hashicorp/aws]  |
                                  |  [google]         |
       |                          |  [azurerm]        |
       v                          +-------------------+
+----------------+                          |
| .terraform/    | <----- downloaded -------+
|   plugins/     |
|   modules/     |
+----------------+
| terraform.lock |  (pins exact versions)
+----------------+
```

You run `init` once when you start a project. You re-run it when you change provider versions or add new providers. You also re-run it when you change your backend (`-reconfigure`).

A typical `init` run:

```
$ terraform init

Initializing the backend...

Successfully configured the backend "s3"! Terraform will automatically
use this backend unless the backend configuration changes.

Initializing provider plugins...
- Finding hashicorp/aws versions matching "~> 5.0"...
- Installing hashicorp/aws v5.31.0...
- Installed hashicorp/aws v5.31.0 (signed by HashiCorp)

Terraform has been successfully initialized!
```

If you forget to run `init` and you try to run `plan` or `apply`, you'll get an error: "this directory has not been initialized." Just run `init`.

### terraform plan

`plan` is the safety net. It shows you what `apply` would do, without actually doing anything.

What `plan` does:

1. **Reads your `.tf` files** (the desired state).
2. **Reads the state file** (the previously-known state).
3. **Optionally refreshes** by calling the cloud APIs to check if reality has drifted from the state file.
4. **Computes a diff** — what needs to be created, what needs to be updated, what needs to be destroyed, what needs to be replaced.
5. **Prints the diff** in a human-readable form with `+`, `-`, `~`, and `+/-` markers.

Picture:

```
+----------------+   +-----------------+   +------------------+
|   .tf files    |   |   state file    |   |   real cloud     |
| (desired)      |   | (last known)    |   | (refresh)        |
+--------+-------+   +--------+--------+   +---------+--------+
         |                    |                      |
         v                    v                      v
                +------------+----------+
                |  TERRAFORM PLAN       |
                |  computes diff:       |
                |  + create N           |
                |  ~ update M           |
                |  - destroy K          |
                |  +/- replace J        |
                +-----------+-----------+
                            |
                            v
                  Plan: 5 to add, 2 to change,
                        1 to destroy.
```

The output of `plan` looks like this (trimmed):

```
$ terraform plan

Terraform will perform the following actions:

  # aws_instance.web will be created
  + resource "aws_instance" "web" {
      + ami                    = "ami-0c55b159cbfafe1f0"
      + instance_type          = "t3.micro"
      + id                     = (known after apply)
      + public_ip              = (known after apply)
      + tags                   = {
          + "Name" = "hello-world"
        }
    }

Plan: 1 to add, 0 to change, 0 to destroy.
```

The `+` means "create." `-` means "destroy." `~` means "update in place." `+/-` (or `-/+`) means "destroy then create" (replace). `(known after apply)` means "Terraform doesn't know this value until the resource is actually created" (e.g., the public IP is assigned by AWS after the instance launches).

You can save the plan to a file:

```
$ terraform plan -out=plan.bin
```

Then later you can apply that exact plan, knowing nobody snuck in a change:

```
$ terraform apply plan.bin
```

This is the recommended workflow for CI/CD pipelines.

### terraform apply

`apply` is the doing. It runs the plan against the cloud's APIs.

If you don't pass a saved plan file, `apply` will compute its own plan first and ask you to confirm:

```
$ terraform apply

Terraform will perform the following actions:
  ...

Plan: 1 to add, 0 to change, 0 to destroy.

Do you want to perform these actions?
  Terraform will perform the actions described above.
  Only 'yes' will be accepted to approve.

  Enter a value: yes
```

You type `yes`. Terraform calls the APIs:

```
aws_instance.web: Creating...
aws_instance.web: Still creating... [10s elapsed]
aws_instance.web: Still creating... [20s elapsed]
aws_instance.web: Creation complete after 28s [id=i-0123456789abcdef0]

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.

Outputs:
public_ip = "54.123.45.67"
```

Done. The EC2 instance exists. The state file has been updated. The output values are printed.

In CI you don't want to type `yes`, so you use:

```
$ terraform apply -auto-approve
```

This skips the confirmation. Use this carefully. Always combine it with a saved plan file in production:

```
$ terraform plan -out=plan.bin
$ terraform apply plan.bin       # plan files don't need -auto-approve
```

That way you saw the diff (in the `plan` step), reviewed it, and then applied exactly what was approved.

### terraform destroy

`destroy` is the opposite of `apply`. It tears down everything in the state file.

```
$ terraform destroy

Terraform will perform the following actions:

  # aws_instance.web will be destroyed
  - resource "aws_instance" "web" {
      - id                     = "i-0123456789abcdef0" -> null
      - public_ip              = "54.123.45.67" -> null
      ...
    }

Plan: 0 to add, 0 to change, 1 to destroy.

Do you really want to destroy all resources?
  Terraform will destroy all your managed infrastructure, as shown above.
  There is no undo. Only 'yes' will be accepted to confirm.

  Enter a value: yes

aws_instance.web: Destroying... [id=i-0123456789abcdef0]
aws_instance.web: Destruction complete after 41s

Destroy complete! Resources: 1 destroyed.
```

That instance is gone. The state file is empty. The cloud bill stops growing.

`destroy` is useful for tearing down throwaway environments (a feature branch's preview env, a staging cluster you no longer need). It is dangerous in production (there is no undo). Some teams use a `prevent_destroy` lifecycle rule on production resources to make sure `destroy` cannot accidentally nuke them.

### A picture of the full lifecycle

```
        +------------+
        |  init      |  download providers
        +-----+------+  set up backend
              |
              v
        +------------+
   +--->|  plan      |  see what would change
   |    +-----+------+
   |          |
   |          v
   |    +------------+
   |    |  apply     |  do it
   |    +-----+------+
   |          |
   |          v
   |    +------------+
   |    | state file |  remembers what was done
   |    +-----+------+
   |          |
   |          | (you edit .tf files)
   +----------+
              |
              v
        +------------+
        |  destroy   |  (eventually) tear down
        +------------+
```

Edit, plan, apply. Edit, plan, apply. Forever. That is the loop.

## Anatomy of a .tf File

Let's walk through a complete, working `.tf` file at ELI5 level. Here it is:

```hcl
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  backend "s3" {
    bucket = "my-tf-state"
    key    = "prod/terraform.tfstate"
    region = "us-east-1"
  }
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_instance" "web" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.micro"

  tags = {
    Name = "hello-world"
  }
}

output "public_ip" {
  value = aws_instance.web.public_ip
}
```

That is a complete, working Terraform file. If you have AWS credentials and an S3 bucket called `my-tf-state`, you can run `terraform init && terraform apply` against this file and you'll get an EC2 instance.

Now let's go block by block.

### The terraform block

```hcl
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  backend "s3" {
    bucket = "my-tf-state"
    key    = "prod/terraform.tfstate"
    region = "us-east-1"
  }
}
```

This is the **terraform block**. It is the meta-block. It tells Terraform itself what it needs.

`required_providers` lists every cloud or service plugin this code depends on. Here we say we want the AWS provider. `source = "hashicorp/aws"` means "the official AWS provider on the Terraform Registry, published by HashiCorp." `version = "~> 5.0"` is a version constraint meaning "any 5.x version, but not 6.0 or higher." The `~>` operator is read as "pessimistic constraint" — it means "pick the newest patch within this minor."

A few common version operators:

- `= 5.0.0` — exactly this version, no other.
- `>= 5.0` — this version or higher.
- `~> 5.0` — any 5.x (pessimistic).
- `~> 5.0.0` — any 5.0.x (more pessimistic).
- `>= 5.0, < 6.0` — explicit range.
- `!= 5.1.0` — anything except this one buggy version.

`backend "s3"` is the **backend** declaration. The backend is where Terraform stores the state file. Here we say S3. The backend takes its own configuration — which bucket, which key (path inside the bucket), which region. We'll talk about backends in much more detail later.

The `terraform` block can also include things like `required_version = ">= 1.5"` to pin which version of Terraform itself can run this code. This is useful when you use new HCL features that older Terraform versions don't understand.

### The provider block

```hcl
provider "aws" {
  region = "us-east-1"
}
```

The **provider block** configures one specific provider. We declared we *need* the AWS provider in the terraform block. Now we *configure* it: which region to use, which credentials to use (usually picked up from environment variables or `~/.aws/credentials`, not put in the file), which assume-role to use, etc.

You can have multiple provider blocks for the same provider with different aliases:

```hcl
provider "aws" {
  region = "us-east-1"
}

provider "aws" {
  alias  = "europe"
  region = "eu-west-1"
}
```

Now resources can pick which provider to use:

```hcl
resource "aws_s3_bucket" "us_data" {
  bucket = "my-data-us"
  # uses default provider (us-east-1)
}

resource "aws_s3_bucket" "eu_data" {
  bucket   = "my-data-eu"
  provider = aws.europe   # uses the aliased provider (eu-west-1)
}
```

This is how you do multi-region deployments in one Terraform config.

### The resource block

```hcl
resource "aws_instance" "web" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.micro"

  tags = {
    Name = "hello-world"
  }
}
```

The **resource block** is where the action is. This is what creates real things in the cloud.

The block syntax is:

```
resource "<TYPE>" "<NAME>" {
  ... attributes ...
}
```

`TYPE` is the resource type (`aws_instance`, `aws_s3_bucket`, `google_compute_instance`, `azurerm_virtual_machine`, `cloudflare_record`, etc.). The first part of the type before the underscore tells you the provider.

`NAME` is your local label for this resource. It's how YOU refer to this specific resource elsewhere in your code. It is NOT the name of the resource in the cloud (that's set by attributes like `tags = { Name = "..." }` or `bucket = "..."`). The name you choose only matters within Terraform.

Inside the block are **attributes**: key-value pairs that configure the resource. `ami` is the Amazon Machine Image ID (which OS image to launch). `instance_type` is the size of the VM. `tags` is a map of key-value pairs to attach to the instance.

Each resource type has different attributes. The provider's documentation lists every attribute, every default, every required one. The AWS instance docs are at registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/instance.

### Resource addresses

Once you declare a resource, you can refer to it elsewhere using its **address**. The address is `<TYPE>.<NAME>`. So `aws_instance.web` is "my AWS instance called web."

Inside the address you can drill into attributes. `aws_instance.web.public_ip` is "the public IP of my AWS instance called web." `aws_instance.web.id` is the instance ID. `aws_instance.web.private_ip` is the private IP.

You also see the address in plan output (`# aws_instance.web will be created`), in state commands (`terraform state show aws_instance.web`), and in error messages.

### The output block

```hcl
output "public_ip" {
  value = aws_instance.web.public_ip
}
```

An **output block** exposes a value. After `apply`, Terraform prints all outputs:

```
Outputs:

public_ip = "54.123.45.67"
```

You can also fetch outputs later with `terraform output public_ip` or `terraform output -json` for machine-readable form.

Outputs are also how modules expose values to their parent module (more on modules later).

### Comments

HCL supports two comment styles:

```hcl
# Single-line comment (preferred)
// Single-line comment (also works)

/*
  Multi-line comment.
  Useful for big blocks of explanatory text.
*/
```

Use `#` for everyday comments. Most teams' style guides standardize on it.

### Whitespace and formatting

HCL is whitespace-flexible but has an enforced canonical format. Run:

```
$ terraform fmt
```

This rewrites all your `.tf` files into the canonical layout: aligned `=` signs, two-space indentation, blank lines between blocks. Most teams run `terraform fmt -check` in CI to enforce it.

## Resources, Data Sources, Modules

Three core kinds of building blocks.

### Resource — the thing you create

We just covered these. `resource "aws_instance" "web" {}` creates and manages a real cloud resource. Terraform owns it. If you delete the block, Terraform destroys the resource.

### Data Source — the thing you read

A **data source** is read-only. It doesn't create anything. It reads existing things from the cloud (or anywhere else) and exposes them as values you can use in your configuration.

```hcl
data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"]   # Canonical's AWS account
  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }
}

resource "aws_instance" "web" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = "t3.micro"
}
```

Now we don't have to hardcode the AMI ID. The `data "aws_ami" "ubuntu"` block queries AWS for the most recent Ubuntu 22.04 AMI. Then we use `data.aws_ami.ubuntu.id` (the address of a data source is `data.<TYPE>.<NAME>`) to pass it into the instance.

Data sources are great for:

- Looking up an existing AMI by name pattern.
- Looking up a VPC's CIDR by tag.
- Looking up an SSL certificate's ARN by domain.
- Reading secrets from Vault.
- Reading a JSON file from S3.
- Pretty much anything where you want to use existing cloud state without having Terraform own it.

Data sources are refreshed every time you run `plan` or `apply`. They are always read-only.

### Module — the reusable bundle

A **module** is a folder full of `.tf` files that you treat as one unit. You call a module like a function:

```hcl
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.5.0"

  name = "my-vpc"
  cidr = "10.0.0.0/16"

  azs             = ["us-east-1a", "us-east-1b", "us-east-1c"]
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]

  enable_nat_gateway = true
  single_nat_gateway = true
}
```

That one block creates a whole VPC, three private subnets, three public subnets, an internet gateway, a NAT gateway, route tables, route table associations, and connects everything. Behind the scenes it uses about 30 different `resource` blocks. You don't see any of that. You just see the module call.

The module's `source` can be:

- **A registry path** — `terraform-aws-modules/vpc/aws` is on the public Terraform Registry.
- **A git URL** — `git::https://github.com/myorg/my-module.git//path/to/module?ref=v1.2.3`
- **A local path** — `./modules/vpc` (relative to the current directory).
- **An HTTP URL** — `https://example.com/module.zip`.
- **An S3 URL** — `s3::https://s3.amazonaws.com/my-bucket/module.zip`.

A module exposes:

- **Inputs** — variables the parent passes in (`name`, `cidr`, etc.).
- **Outputs** — values the parent can read (`module.vpc.vpc_id`, `module.vpc.private_subnets`).

Modules are the main way teams keep DRY (Don't Repeat Yourself). One team writes a great VPC module. Every other team uses it. When the VPC module gets a new feature, everyone gets it on their next `terraform init -upgrade`.

There are thousands of modules on the Terraform Registry. The `terraform-aws-modules` namespace alone has dozens of high-quality, well-maintained modules. Don't write your own VPC, EKS, RDS, ALB, IAM, or VPN module from scratch — use the existing ones first, and only fork if you really need something custom.

### A picture of the three kinds

```
+------------------+   creates and owns
|    resource      | -----------------------> [real cloud thing]
+------------------+

+------------------+   reads (does not own)
|   data source    | <----------------------- [real cloud thing]
+------------------+

+------------------+   contains many resources/data/modules
|     module       | -----------------------> { resource, resource,
+------------------+                            data, module... }
```

A typical real-world `.tf` file uses all three: data sources to look things up, modules to encapsulate big chunks of common patterns, and resources for the few things that are specific to your project.

## State — The Most Important Concept

This is the section to read twice. State is where most Terraform pain comes from. Understand state and you understand Terraform.

### What is the state file?

The state file is a JSON document, usually named `terraform.tfstate`. It is Terraform's notebook — the record of every resource Terraform manages.

A tiny excerpt:

```json
{
  "version": 4,
  "terraform_version": "1.6.4",
  "serial": 12,
  "lineage": "abcd-efgh-1234",
  "outputs": {
    "public_ip": {
      "value": "54.123.45.67",
      "type": "string"
    }
  },
  "resources": [
    {
      "module": "",
      "mode": "managed",
      "type": "aws_instance",
      "name": "web",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "schema_version": 1,
          "attributes": {
            "id": "i-0123456789abcdef0",
            "ami": "ami-0c55b159cbfafe1f0",
            "instance_type": "t3.micro",
            "public_ip": "54.123.45.67",
            "private_ip": "10.0.1.15",
            "tags": {
              "Name": "hello-world"
            }
          }
        }
      ]
    }
  ]
}
```

It's the full attribute map of every resource Terraform created or imported. It also has metadata: the schema version, the lineage (a unique ID for this state's "history"), the serial (incremented every write).

### What does state actually do?

Three big jobs:

1. **Mapping.** It maps resources in your `.tf` files (`aws_instance.web`) to real cloud objects (`i-0123456789abcdef0`). Without this map, Terraform can't tell that `aws_instance.web` IS that EC2 instance.
2. **Caching.** It caches all the attributes of each resource so plan can compare without making API calls (or, if `-refresh-only` is used, after refreshing).
3. **Performance.** Some clouds have slow APIs. State means Terraform can compute a plan against the last-known state without re-querying everything.

### Why it must NEVER be edited by hand

The state file format has internal invariants. The schema version must match. The lineage must be consistent. The serial must increment on every write. If you hand-edit the JSON and break one of these, Terraform will refuse to read it.

If you hand-edit the JSON and DON'T break those invariants, you might end up with state that says one thing and reality saying another, which causes Terraform to make weird decisions on the next plan (delete things that exist, recreate things that already exist correctly, etc.).

There are official commands for editing state safely:

- `terraform state list` — list all resources in state.
- `terraform state show <addr>` — show one resource's full state.
- `terraform state mv <src> <dst>` — rename a resource's address.
- `terraform state rm <addr>` — remove a resource from state without destroying it.
- `terraform state replace-provider` — change the provider source.

Use these. Don't open the JSON in vim.

### Why it must NEVER be checked into git

Two reasons:

1. **Secrets.** State files often contain sensitive data: database passwords, API tokens, private keys. Even if you mark variables as `sensitive` in your `.tf` files, the state file stores the full unencrypted value (the cloud needs the actual value to provision the resource, and Terraform stores what it sent). Putting state in git means putting secrets in git.

2. **Concurrency.** If two team members run `terraform apply` at the same time on the same state file, you'll get conflicts. Git is not a concurrency-control system. Two parallel applies that each commit a different state will produce a merge conflict that has no good resolution.

The right answer: **remote state with locking.**

### Remote state backends

A **backend** is where Terraform stores the state file. The default is `local` — which means a `terraform.tfstate` file in the current directory. This is fine for solo learning and toy projects. Anything else, use remote state.

The big remote backends:

- **S3** (with DynamoDB for locking) — most common for AWS shops.
- **GCS** (Google Cloud Storage) — for GCP shops, has built-in locking.
- **Azure Blob Storage** — for Azure shops, has built-in locking.
- **Terraform Cloud / Terraform Enterprise** (HashiCorp's managed service) — has UI, history, RBAC, and is free for small teams.
- **Consul** — distributed key-value store, supports locking.
- **HTTP** — generic REST endpoint, useful if you have your own state service.

A typical S3 backend config:

```hcl
terraform {
  backend "s3" {
    bucket         = "my-tf-state"
    key            = "prod/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "my-tf-lock"
    encrypt        = true
  }
}
```

The `bucket` is your S3 bucket. The `key` is the path inside the bucket — usually one path per environment (`prod/`, `staging/`, `dev/`). The `dynamodb_table` is for locking (more on that next). `encrypt = true` enables server-side encryption.

### State locking

If two people run `apply` against the same state at the same time, bad things happen. State A overwrites state B. Or vice versa. Or you end up with state that thinks two resources exist when only one does, or vice versa.

So Terraform has **locking**. Before any operation that writes state (apply, destroy, state mv, state rm, etc.), Terraform tries to acquire a lock. If another operation already holds the lock, the new one waits or fails.

How locking works depends on the backend:

- **S3** uses **DynamoDB** for locking (you create a small DynamoDB table with a `LockID` primary key). Terraform writes a row when it acquires the lock and deletes the row when it releases it. Other operations check this row.
- **GCS** has native object-locking support. No separate lock table needed.
- **Azure Blob** uses lease-based locking on the state blob itself.
- **Terraform Cloud** has its own lock system, with UI to see who's holding it.
- **Consul** uses Consul's session-based locks.

When a lock is held you'll see:

```
Acquiring state lock. This may take a few moments...
Error: Error acquiring the state lock

Error message: ConditionalCheckFailedException: The conditional request failed
Lock Info:
  ID:        abc123-def456
  Path:      my-tf-state/prod/terraform.tfstate
  Operation: OperationTypeApply
  Who:       alice@bigcorp.com
  Version:   1.6.4
  Created:   2024-04-27 14:32:11 UTC
```

That tells you alice is currently applying. Wait for her, or coordinate.

If a lock gets stuck (process crashed, network interrupted), you can force-unlock:

```
$ terraform force-unlock abc123-def456
```

But ONLY do this if you are CERTAIN no other apply is actually running. Force-unlocking an active apply will corrupt state.

### State file picture

```
                 +-------------------+
                 |  Your .tf files   |
                 |  (in git)         |
                 +---------+---------+
                           |
                           | terraform plan/apply
                           v
+----------------+         +---------------+        +---------------+
|   You          |  reads  |  Terraform    | reads  | Cloud APIs    |
|   (or CI)      |-------->|  CLI          |------->| (AWS/GCP/etc) |
|                |  state  |               | calls  |               |
+----------------+         +-------+-------+        +---------------+
                                   |
                                   | reads/writes state
                                   v
                           +-------+-------+        +---------------+
                           | Remote backend|<------>| Lock table    |
                           | (S3/GCS/TFC)  |  lock  | (DynamoDB,    |
                           +---------------+        |  GCS native,  |
                           | terraform     |        |  TFC native)  |
                           | .tfstate      |        +---------------+
                           +---------------+
```

The state file lives in the remote backend. Your local copy is a transient cache during a run. The lock table prevents concurrent runs.

### .tfstate.backup

Every time Terraform writes state, it copies the previous state to `terraform.tfstate.backup` (locally) or to a versioned object (in remote backends with versioning, like S3 with `versioning = enabled`). If something goes catastrophically wrong, you can roll back to the backup.

Always enable versioning on your S3 state bucket. Always.

### A real-world recommendation

For a serious project, your state setup looks like:

- **Backend**: S3 (for AWS teams), GCS (for GCP), or Terraform Cloud.
- **Bucket versioning**: enabled.
- **Server-side encryption**: KMS or AES-256.
- **Locking**: DynamoDB (for S3), native (for GCS, Azure, TFC).
- **Bucket policy**: only the Terraform service role can write; team can read.
- **One state file per environment**: `prod/`, `staging/`, `dev/`, `feature-foo/`.

This is the bare minimum for a team. Don't skimp.

## Variables and Outputs

### Variables

A **variable** is an input to your Terraform configuration. You declare it with a `variable` block:

```hcl
variable "instance_type" {
  description = "EC2 instance type for the web server"
  type        = string
  default     = "t3.micro"
}

variable "instance_count" {
  type    = number
  default = 1
}

variable "tags" {
  type = map(string)
  default = {
    Environment = "prod"
    Owner       = "platform-team"
  }
}
```

Then you use it elsewhere with `var.<NAME>`:

```hcl
resource "aws_instance" "web" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = var.instance_type
  count         = var.instance_count

  tags = var.tags
}
```

### Variable types

The `type` attribute restricts what values the variable can take. Common types:

- `string` — `"hello"`.
- `number` — `42` or `3.14`.
- `bool` — `true` or `false`.
- `list(string)` — `["a", "b", "c"]`.
- `set(string)` — `["a", "b", "c"]` but unordered and unique.
- `map(string)` — `{ key = "value", other = "stuff" }`.
- `object({ name = string, age = number })` — like a struct.
- `tuple([string, number, bool])` — like a list but each position has a fixed type.
- `any` — anything goes (escape hatch, avoid).

You can also use `optional` and `nullable`:

```hcl
variable "config" {
  type = object({
    name    = string
    version = optional(string, "v1.0")
    enabled = optional(bool, true)
  })
}
```

### Variable validation

You can add validation rules:

```hcl
variable "instance_type" {
  type    = string
  default = "t3.micro"

  validation {
    condition     = contains(["t3.micro", "t3.small", "t3.medium"], var.instance_type)
    error_message = "instance_type must be one of t3.micro, t3.small, or t3.medium."
  }
}
```

Now if someone passes `t3.huge`, Terraform errors out at plan time with your custom message.

### How values get into variables (the precedence order)

A single variable can have its value set in many places. Terraform uses this priority order, highest priority wins:

1. **Command-line `-var`**: `terraform apply -var="instance_type=t3.large"`.
2. **Command-line `-var-file`**: `terraform apply -var-file=prod.tfvars`.
3. **`*.auto.tfvars` files** (loaded automatically, alphabetical order).
4. **`terraform.tfvars`** file (loaded automatically).
5. **Environment variables** named `TF_VAR_<name>`: `export TF_VAR_instance_type=t3.large`.
6. **Variable default** in the `variable` block.

If a variable has no default and no value is provided, Terraform prompts you interactively (which is fine for ad-hoc, but breaks CI).

A `terraform.tfvars` looks like:

```hcl
instance_type = "t3.large"
instance_count = 3
tags = {
  Environment = "prod"
  Owner       = "platform-team"
}
```

You usually have one `terraform.tfvars` per environment, or one `prod.tfvars`, `staging.tfvars`, `dev.tfvars` set you pass in with `-var-file`.

### Sensitive variables

Mark variables sensitive to keep them out of CLI output:

```hcl
variable "db_password" {
  type      = string
  sensitive = true
}
```

When you run plan or apply, the value will be replaced with `(sensitive value)` in the output. Note: the value is still in the state file in plaintext. Sensitive only hides it from logs, not from state. For real secrets, fetch them from Vault or AWS Secrets Manager via a data source.

### Outputs

We saw outputs already:

```hcl
output "public_ip" {
  value       = aws_instance.web.public_ip
  description = "Public IP of the web server"
}
```

Outputs can also be sensitive:

```hcl
output "db_endpoint" {
  value     = aws_db_instance.main.endpoint
  sensitive = true
}
```

A sensitive output won't be printed by `terraform apply`, but you can still fetch it explicitly with `terraform output db_endpoint`.

Outputs from one module become accessible in the parent as `module.<NAME>.<OUTPUT>`:

```hcl
module "vpc" {
  source = "./modules/vpc"
}

resource "aws_instance" "web" {
  subnet_id = module.vpc.public_subnet_id
}
```

This is the main way modules communicate.

### Output for scripting

```
$ terraform output
public_ip = "54.123.45.67"

$ terraform output public_ip
"54.123.45.67"

$ terraform output -raw public_ip
54.123.45.67

$ terraform output -json
{
  "public_ip": {
    "sensitive": false,
    "type": "string",
    "value": "54.123.45.67"
  }
}
```

The `-json` form is the canonical way to consume outputs in shell scripts:

```
$ ip=$(terraform output -raw public_ip)
$ ssh ubuntu@$ip
```

## Locals and Expressions

### Locals

A `locals` block defines named values you can use elsewhere. They're like variables, but computed inside the config (not passed in from outside).

```hcl
locals {
  environment = "prod"
  region      = "us-east-1"

  common_tags = {
    Environment = local.environment
    Region      = local.region
    ManagedBy   = "Terraform"
  }

  full_name = "${local.environment}-web-${local.region}"
}

resource "aws_instance" "web" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = "t3.micro"
  tags          = local.common_tags
}
```

You use `local.<NAME>` (singular!) to reference them. You declare them in `locals` (plural). This trips people up.

Locals are great for:

- Tags you reuse across many resources.
- Computed names (`"${env}-${region}-web"`).
- Conditional values (`local.use_nat = var.environment == "prod" ? true : false`).
- Anything you find yourself typing more than twice.

### Expressions

HCL has an expression language. Things you can do inside `${...}` or attribute values:

**String interpolation:**

```hcl
name = "web-${var.environment}-${count.index}"
```

**Math:**

```hcl
disk_size = var.base_size * 2 + 10
```

**Comparison and boolean:**

```hcl
is_prod = var.environment == "prod"
needs_backup = var.environment == "prod" || var.always_backup
```

**Ternary (conditional):**

```hcl
instance_type = var.environment == "prod" ? "t3.large" : "t3.micro"
```

**Lists and maps:**

```hcl
servers = ["web1", "web2", "web3"]
config  = { region = "us-east-1", env = "prod" }
```

**Indexing:**

```hcl
first  = local.servers[0]   # "web1"
region = local.config["region"]   # "us-east-1"
```

**Functions:**

HCL has many built-in functions. Categories include:

- **String**: `upper()`, `lower()`, `format()`, `join()`, `split()`, `replace()`, `trim()`.
- **Numeric**: `min()`, `max()`, `abs()`, `ceil()`, `floor()`.
- **Collection**: `length()`, `keys()`, `values()`, `merge()`, `concat()`, `flatten()`.
- **Encoding**: `jsonencode()`, `jsondecode()`, `yamlencode()`, `yamldecode()`, `base64encode()`.
- **Filesystem**: `file()`, `templatefile()`, `fileexists()`.
- **Date/Time**: `timestamp()`, `formatdate()`.
- **Hash**: `sha1()`, `sha256()`, `md5()`.
- **IP/Network**: `cidrsubnet()`, `cidrhost()`, `cidrnetmask()`.
- **Type**: `tostring()`, `tonumber()`, `tolist()`, `toset()`, `tomap()`.

A common pattern is `templatefile()`:

```hcl
resource "aws_instance" "web" {
  user_data = templatefile("${path.module}/cloud-init.tpl", {
    hostname = "web-${var.environment}"
    region   = var.region
  })
}
```

This reads a template file and substitutes the variables. Very clean.

**For expressions:**

```hcl
locals {
  upper_names = [for s in var.names : upper(s)]
  by_id       = { for u in var.users : u.id => u.name }
}
```

A `for` expression iterates over a collection and produces a new one.

**Splat expressions:**

```hcl
locals {
  all_ids = aws_instance.web[*].id   # array of all IDs from a counted resource
}
```

The `[*]` operator extracts an attribute from every element of a list.

### path.module and path.root

Two useful built-in references:

- `path.module` — directory of the current module.
- `path.root` — directory of the root module (where you ran terraform).
- `path.cwd` — current working directory.

You use these for `templatefile()`, `file()`, etc., to make paths portable:

```hcl
user_data = file("${path.module}/scripts/bootstrap.sh")
```

## for_each and count

You often want multiple copies of a resource. Three buckets, ten EC2 instances, one DNS record per region. Terraform has two ways to do this: `count` and `for_each`. They look similar but behave very differently.

### count — list-based, indexed

```hcl
resource "aws_instance" "web" {
  count         = 3
  ami           = data.aws_ami.ubuntu.id
  instance_type = "t3.micro"

  tags = {
    Name = "web-${count.index}"
  }
}
```

This creates 3 instances, addressable as `aws_instance.web[0]`, `aws_instance.web[1]`, `aws_instance.web[2]`.

`count.index` is available inside the block.

### count's gotcha

What happens if you go from `count = 3` to `count = 2`? Terraform destroys `aws_instance.web[2]`.

What if you go from `count = 3` to `count = 3` but you reorder which instance you wanted to remove? You can't. With count, the only way to remove the *middle* one is to remove the last index. If you have web[0], web[1], web[2] and you want to remove web[1], you can't. Going to count=2 destroys web[2].

This is the classic count problem: list-indexed resources are fragile to add/remove from the middle.

### for_each — map-based, keyed

```hcl
resource "aws_instance" "web" {
  for_each = toset(["alice", "bob", "carol"])

  ami           = data.aws_ami.ubuntu.id
  instance_type = "t3.micro"

  tags = {
    Name = "web-${each.key}"
  }
}
```

This creates 3 instances, addressable as `aws_instance.web["alice"]`, `aws_instance.web["bob"]`, `aws_instance.web["carol"]`.

Inside the block: `each.key` is the key, `each.value` is the value (in a map; in a set, key and value are the same).

### for_each over a map

```hcl
locals {
  servers = {
    web = {
      type = "t3.medium"
      ami  = "ami-aaa"
    }
    db = {
      type = "t3.large"
      ami  = "ami-bbb"
    }
  }
}

resource "aws_instance" "all" {
  for_each = local.servers

  ami           = each.value.ami
  instance_type = each.value.type

  tags = {
    Name = each.key
  }
}
```

This creates `aws_instance.all["web"]` and `aws_instance.all["db"]`, each with its own type and ami.

### Why for_each is usually better

If you go from `{ alice, bob, carol }` to `{ alice, carol }`, Terraform destroys ONLY `aws_instance.web["bob"]`. Alice and Carol are untouched. The keys are stable.

With count, removing bob from the middle is much messier (you'd end up with carol's content moving to bob's index, etc.).

The general rule: **prefer for_each unless you really do have a positional list**. If you're indexing by name, role, region, environment, or anything string-y, use for_each.

### A picture

```
   count = 3                     for_each = ["alice","bob","carol"]
   +---------------+             +-----------------------+
   |  web[0]       |             |  web["alice"]         |
   |  web[1]       |             |  web["bob"]           |
   |  web[2]       |             |  web["carol"]         |
   +---------------+             +-----------------------+

   Remove middle?                Remove middle?
   count = 2 destroys web[2]     remove "bob" from set,
   (NOT web[1] you wanted!)      destroys ONLY web["bob"]
```

### dynamic blocks

Sometimes you want to repeat a sub-block (like `ingress` rules in a security group) instead of a whole resource. Use `dynamic`:

```hcl
resource "aws_security_group" "web" {
  name = "web"

  dynamic "ingress" {
    for_each = var.allowed_ports
    content {
      from_port   = ingress.value
      to_port     = ingress.value
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
    }
  }
}
```

`dynamic "ingress"` says "make this many `ingress` blocks, one per element in `var.allowed_ports`."

## Provisioners (And Why You Should Avoid Them)

A **provisioner** runs a command after a resource is created (or before it's destroyed). The two common ones are `local-exec` and `remote-exec`.

```hcl
resource "aws_instance" "web" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = "t3.micro"

  provisioner "local-exec" {
    command = "echo ${self.public_ip} >> ips.txt"
  }

  provisioner "remote-exec" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y nginx",
    ]

    connection {
      type        = "ssh"
      user        = "ubuntu"
      private_key = file("~/.ssh/id_rsa")
      host        = self.public_ip
    }
  }
}
```

`local-exec` runs on the machine running Terraform. `remote-exec` SSHs into the resource and runs commands on it.

### Why you should avoid them

The official Terraform docs literally start the provisioners section with: "Provisioners should be a last resort." Reasons:

1. **They're not part of the state model.** If a provisioner runs and partially completes, Terraform doesn't really know. You can end up with state that says "created" but the box wasn't actually provisioned.
2. **They're imperative.** Terraform is supposed to be declarative. Provisioners drag you back into "do this then this then this."
3. **They run only once at create time** (by default). If your Ansible playbook changes, Terraform won't notice. The resource is unchanged from Terraform's view.
4. **They tie infra to config.** A common anti-pattern is using `remote-exec` to install software — but software changes much more often than infra, and you really don't want to recreate a whole VM every time you patch a package.

### What to use instead

- **For initial OS configuration** (install packages, create users): use `user_data` (cloud-init) on EC2/GCP/Azure VMs. This is a tiny script that runs on first boot. It's part of the resource definition, so Terraform sees it.
- **For ongoing config management** (patches, app deploys, daily drift): use Ansible / Chef / Puppet / SaltStack / `cs ramp-up ansible-eli5`. These tools are built for "make this box look like this" and they're idempotent.
- **For Kubernetes apps**: deploy through k8s itself (Helm, kubectl, Argo CD). Don't try to deploy apps via Terraform provisioners.
- **For one-time setup that genuinely is part of the resource**: `null_resource` with a `local-exec` and a `triggers` block.

### When provisioners ARE okay

- Bootstrapping a brand-new ssh key or initial password.
- Running a simple notification (push to Slack on apply).
- Setup that's truly one-shot and won't change.

Even then, prefer `null_resource` to attaching the provisioner to a real resource:

```hcl
resource "null_resource" "notify" {
  triggers = {
    instance_id = aws_instance.web.id
  }

  provisioner "local-exec" {
    command = "curl -X POST https://hooks.slack.com/... -d 'New web server: ${aws_instance.web.public_ip}'"
  }
}
```

This way the provisioner is a separate resource — its lifecycle and triggers are explicit, and a provisioner failure doesn't leave your real resource in a half-state.

## Workspaces

A **workspace** is a logical environment name. Terraform supports having multiple state files in one config, picked by workspace name.

```
$ terraform workspace list
* default

$ terraform workspace new staging
Created and switched to workspace "staging"!

$ terraform workspace new prod
Created and switched to workspace "prod"!

$ terraform workspace list
  default
  staging
* prod

$ terraform workspace select staging
Switched to workspace "staging".

$ terraform workspace show
staging
```

The current workspace name is exposed as `terraform.workspace`:

```hcl
resource "aws_instance" "web" {
  instance_type = terraform.workspace == "prod" ? "t3.large" : "t3.micro"

  tags = {
    Environment = terraform.workspace
  }
}
```

Behind the scenes, each workspace is a separate state file. With S3 backend, they live in `<bucket>/env:/<workspace>/<key>`. With local backend, they're in `terraform.tfstate.d/<workspace>/`.

### Are workspaces good for environments?

Officially: **no, not really**. The Terraform docs explicitly recommend AGAINST using workspaces to separate prod from staging from dev. Why?

1. **Same code base.** Workspaces share the same `.tf` files. If you have prod-only resources, you have to gate them with `count` and `terraform.workspace` checks. Gross.
2. **Same providers.** All workspaces use the same provider config. Hard to have different AWS accounts per environment.
3. **No isolation.** A bad apply in dev workspace can affect the local Terraform cache that prod uses.
4. **Hard to review.** A PR that touches prod-or-staging-or-dev is harder to scope.

### What to use instead

The recommended pattern is **separate root modules per environment**:

```
infra/
  prod/
    main.tf
    variables.tf
    backend.tf  -> points to prod state
  staging/
    main.tf
    variables.tf
    backend.tf  -> points to staging state
  dev/
    main.tf
    variables.tf
    backend.tf  -> points to dev state
  modules/
    vpc/
    eks/
    rds/
```

Each environment has its own root module, its own state, its own backend, its own providers. They share modules but otherwise are isolated. A PR that changes only `prod/` is reviewable as "prod-only." A bad apply in dev doesn't risk anything else.

For more sophistication, **Terragrunt** (a thin wrapper around Terraform) helps keep these per-env root modules DRY without sharing state. See `cs iac terragrunt`.

If you're using **Terraform Cloud**, you don't need workspaces in this sense at all — TFC has its own first-class concept of "workspace" that's much closer to "an environment with its own state."

### When workspaces ARE okay

- Quick toy or learning projects.
- Personal feature branches that need their own state temporarily.
- Per-developer dev environments.

Otherwise, separate root modules.

## Drift — When Reality Doesn't Match State

**Drift** is when the real cloud state has diverged from what Terraform's state file says. This happens because somebody did something outside of Terraform — clicked something in the console, ran an aws-cli command, edited a security group, etc.

### Detecting drift

`terraform plan` will show drift. Terraform reads the state file, refreshes by calling the cloud APIs, and compares.

```
$ terraform plan
aws_instance.web: Refreshing state... [id=i-0123456789abcdef0]

Note: Objects have changed outside of Terraform

Terraform detected the following changes made outside of Terraform since
the last "terraform apply":

  # aws_instance.web has been changed
  ~ resource "aws_instance" "web" {
        id             = "i-0123456789abcdef0"
      ~ instance_type  = "t3.micro" -> "t3.large"
      ...
    }
```

Somebody resized the instance in the AWS console. Terraform noticed.

Now Terraform has a choice:

- If your `.tf` says `instance_type = "t3.micro"`, the next `apply` will revert it back to `t3.micro` (Terraform fights drift).
- If you actually wanted `t3.large` going forward, change the `.tf` to match and apply (which becomes a no-op).

### terraform apply -refresh-only

Sometimes you want to update the state file to match reality, without changing reality. Use:

```
$ terraform apply -refresh-only
```

This refreshes state from cloud reality. No changes to cloud are made. Now your state matches reality. Useful when you've intentionally made out-of-band changes and want them reflected in state.

(`terraform refresh` is the older form. It's deprecated in favor of `apply -refresh-only`.)

### terraform import

What if you have an existing resource that Terraform doesn't know about, and you want to start managing it with Terraform?

You write the `.tf` block first:

```hcl
resource "aws_instance" "web" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.micro"
  # ... fill in to match the existing instance
}
```

Then you import:

```
$ terraform import aws_instance.web i-0123456789abcdef0
```

Terraform calls the cloud API, fetches all the attributes of `i-0123456789abcdef0`, and writes them into the state file at the address `aws_instance.web`. Now Terraform knows about it.

The next `plan` will show whatever differences exist between your `.tf` (what you wrote) and the actual resource (what was imported). You iterate on your `.tf` until plan shows no changes — at that point, Terraform's view exactly matches the real resource.

In Terraform 1.5+, you can also use **import blocks** in your `.tf`:

```hcl
import {
  to = aws_instance.web
  id = "i-0123456789abcdef0"
}

resource "aws_instance" "web" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.micro"
}
```

This makes import declarative and reviewable in PRs. After applying, you can remove the `import` block.

### terraform plan -generate-config-out

Terraform 1.5+ can also generate a starter `.tf` file from a real resource:

```
$ terraform plan -generate-config-out=generated.tf
```

This works in tandem with `import` blocks. Terraform reads the resource and writes a `.tf` block that matches. You then clean it up.

### Picture: drift detection

```
+----------------+               +------------------+
|  state file    |               |  real cloud      |
| (last known)   |               | (current)        |
+--------+-------+               +---------+--------+
         |                                 |
         |          terraform plan         |
         |          (refresh)              |
         +-------------+-------------------+
                       |
                       v
              +--------+--------+
              |   compare       |
              +--------+--------+
                       |
        +--------------+---------------+
        |                              |
        v                              v
   +--------+                     +---------+
   | match  |                     | drift!  |
   +--------+                     +----+----+
                                       |
                              +--------+--------+
                              | shown in plan   |
                              | output          |
                              +-----------------+
```

## Common Errors

Here are real Terraform errors you will eventually see, with the exact message and the canonical fix.

### "Error: Backend configuration changed"

```
Error: Backend configuration changed

A change in the backend configuration has been detected, which may require
migrating existing state.
```

You changed the `backend` block (different bucket, different region, different DynamoDB table). Terraform won't auto-migrate state.

**Fix**: Run `terraform init -reconfigure` (start fresh, ignore old state) or `terraform init -migrate-state` (copy state from the old backend to the new one). Choose carefully — `-reconfigure` will lose track of resources unless you re-import them.

### "Error acquiring the state lock"

```
Error: Error acquiring the state lock

Lock Info:
  ID:        abc-123-def
  Path:      my-tf-state/prod/terraform.tfstate
  Operation: OperationTypeApply
  Who:       alice@bigcorp.com
```

Another apply is in progress, OR a previous apply crashed and left a stale lock.

**Fix**: If alice is really applying right now, wait for her. If you're sure no apply is running (process crashed, network died), force-unlock:

```
$ terraform force-unlock abc-123-def
```

Use with extreme caution.

### "Error: Provider produced inconsistent result after apply"

```
Error: Provider produced inconsistent result after apply

When applying changes to aws_instance.web, provider "registry.terraform.io/
hashicorp/aws" produced an unexpected new value: ...
```

Usually a provider bug, OR Terraform's view of the resource doesn't match what the cloud actually returned. Common causes:

- Provider version is buggy (try upgrading or pinning down).
- Cloud API returned eventually-consistent data (try `apply` again — usually resolves itself).
- A new provider feature requires a newer schema than your state has.

**Fix**: Try `terraform apply -refresh-only` to refresh state. Try upgrading the provider (`terraform init -upgrade`). Check the provider's GitHub issues for known bugs.

### "InvalidParameterValue: Value (...) for parameter ami is invalid"

```
Error: Error launching source instance: InvalidAMIID.NotFound:
The image id '[ami-0c55b159cbfafe1f0]' does not exist
```

The AMI ID doesn't exist in the region you're targeting. AMI IDs are region-specific.

**Fix**: Use a `data "aws_ami"` block to look up the AMI by name pattern, not by hard-coding the ID. Or hard-code the right ID per region.

### "timeout while waiting for state to become 'running'"

```
Error: error waiting for EC2 Instance (i-0123) to become ready:
timeout while waiting for state to become 'running'
```

The instance launched but didn't become `running` within the expected time. Could be:

- AWS capacity issue.
- The instance got stuck in `pending` (rare).
- A subnet, IAM, or security-group issue.

**Fix**: Check the instance in the AWS console, see why it's stuck. Sometimes just `terraform apply` again works (transient AWS issue).

### "Module not installed"

```
Error: Module not installed

This module is not yet installed. Run "terraform init" to install all
modules required by this configuration.
```

You added a `module` block but didn't run `init`.

**Fix**: `terraform init`. If you upgraded a module's version, `terraform init -upgrade`.

### "value depends on resource attributes that cannot be determined until apply"

```
Error: Invalid for_each argument

The "for_each" value depends on resource attributes that cannot be determined
until apply, so Terraform cannot predict how many instances will be created.
```

You used a value computed at apply time (like an ID) inside `count` or `for_each`. Those have to be known at plan time.

**Fix**: Refactor so the value is known. Use `-target` for a two-stage apply (apply the dependency first, then the dependent), or restructure your code so the count/for_each value is literal at plan time.

### "Provider configuration not present"

```
Error: Provider configuration not present

To work with aws_instance.web (orphan) its original provider configuration
at provider["registry.terraform.io/hashicorp/aws"].europe is required, but
it has been removed.
```

You removed an aliased provider but a resource that used it still exists in state.

**Fix**: Restore the provider block, or `terraform state rm` the orphaned resource (after destroying it manually if needed).

### "Cycle detected"

```
Error: Cycle: aws_security_group.a, aws_security_group.b
```

Resource A's config references B, and B references A. Terraform's dependency graph is a DAG (Directed Acyclic Graph) — no cycles allowed.

**Fix**: Break the cycle. Often this means using a separate `aws_security_group_rule` resource (which doesn't form the cycle) instead of inline `ingress`/`egress` blocks.

### "Unsupported argument" / "Reference to undeclared resource"

```
Error: Reference to undeclared resource

  on main.tf line 12, in resource "aws_instance" "web":
  12:   subnet_id = aws_subnet.main.id

A managed resource "aws_subnet" "main" has not been declared in the root module.
```

Typo in the resource address, or you removed/renamed a resource without updating references.

**Fix**: Read the error carefully. The line number tells you exactly where. Either declare the missing resource, or fix the reference.

### "Error: Failed to query available provider packages"

```
Error: Failed to query available provider packages

Could not retrieve the list of available versions for provider hashicorp/aws:
could not connect to registry.terraform.io
```

Network problem reaching the Terraform Registry.

**Fix**: Check your internet. Check proxy settings. If your CI is air-gapped, set up a private mirror with `terraform providers mirror`.

### "Error: Unsupported argument" (typo in attribute)

```
Error: Unsupported argument

  on main.tf line 4, in resource "aws_instance" "web":
   4:   instnace_type = "t3.micro"

An argument named "instnace_type" is not expected here. Did you mean "instance_type"?
```

Typo. Terraform tells you what it should be.

**Fix**: Fix the typo. (`terraform fmt` won't catch this; `terraform validate` will.)

### "Error: Resource already managed by Terraform"

```
Error: Resource already managed by Terraform

The resource "aws_instance.web" is already managed by Terraform under
"aws_instance.web". To import to a different address, you'll need to ...
```

You tried to `terraform import` a resource that's already in state.

**Fix**: It's already imported. Run `terraform plan` to see if the existing state is correct.

### Picture: error → root cause map

```
Error message              ->   Root cause              ->   Fix
--------------------------------------------------------------------
"Backend changed"          ->   you changed backend     ->   init -reconfigure
"State lock"               ->   another apply or stale  ->   wait or force-unlock
"Inconsistent result"      ->   provider bug / eventual ->   re-apply or upgrade
"AMI invalid"              ->   wrong region            ->   data source for AMI
"Module not installed"     ->   forgot init             ->   init
"value depends on..."      ->   apply-time in for_each  ->   restructure
"Cycle detected"           ->   A->B->A in graph        ->   break with separate resource
"Reference to undeclared"  ->   typo / removed resource ->   fix reference
"Provider not present"     ->   orphaned aliased res    ->   restore provider or rm state
"Unsupported argument"     ->   typo                    ->   spelling
"Already managed"          ->   already imported        ->   nothing to do
```

## Hands-On

Time to run real `terraform` commands. You will need:

- Terraform installed (`brew install terraform` on macOS, or download from terraform.io).
- Cloud credentials configured (for AWS: `aws configure`).
- A small `.tf` file to play with (or use the AWS one above).

For each command below, type the part after the `$` and press Enter.

### Experiment 1: Check the version

```
$ terraform version
Terraform v1.6.4
on darwin_amd64
+ provider registry.terraform.io/hashicorp/aws v5.31.0
```

The version of Terraform itself, plus any providers cached in this directory.

### Experiment 2: Format your code

```
$ terraform fmt
main.tf

$ terraform fmt -check
```

`fmt` rewrites `.tf` files to canonical formatting. It prints which files it changed. `-check` returns non-zero if any files would change (used in CI to enforce formatting).

### Experiment 3: Validate syntax

```
$ terraform validate
Success! The configuration is valid.
```

Checks that your `.tf` files parse and the references resolve. Doesn't call any APIs. Doesn't need init? Actually it does — it needs providers loaded to know which attributes are valid.

### Experiment 4: Initialize the project

```
$ terraform init

Initializing the backend...
Initializing provider plugins...
- Finding hashicorp/aws versions matching "~> 5.0"...
- Installing hashicorp/aws v5.31.0...

Terraform has been successfully initialized!
```

First step in any new project. Downloads providers, sets up backend.

### Experiment 5: Re-init after provider version change

```
$ terraform init -upgrade

Upgrading modules...
- Finding hashicorp/aws versions matching "~> 5.0"...
- Installing hashicorp/aws v5.32.0...
- Installed hashicorp/aws v5.32.0 (signed by HashiCorp)
```

`-upgrade` tells Terraform to ignore the lockfile and grab the latest matching versions. Updates the lockfile. Use sparingly — pinned versions are good.

### Experiment 6: Re-init after backend change

```
$ terraform init -reconfigure

Initializing the backend...

Successfully configured the backend "s3"!
```

`-reconfigure` discards the old backend connection and starts fresh. Use after changing your `backend` block.

### Experiment 7: See what would change (plan)

```
$ terraform plan

Terraform used the selected providers to generate the following execution plan...

Plan: 1 to add, 0 to change, 0 to destroy.
```

Always run plan first. Read every line. Make sure the diff is what you expect.

### Experiment 8: Save a plan to a file

```
$ terraform plan -out=plan.bin

Plan: 1 to add, 0 to change, 0 to destroy.

Saved the plan to: plan.bin
```

Saved plans capture the exact state at plan time. Apply them later to ensure no surprise drift. Use this in CI.

### Experiment 9: Apply a saved plan

```
$ terraform apply plan.bin

aws_instance.web: Creating...
aws_instance.web: Creation complete after 28s [id=i-0123456789abcdef0]

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.
```

When you pass a plan file, Terraform doesn't ask for confirmation (the plan was already approved). Don't pass `-auto-approve` with a plan file.

### Experiment 10: Apply without confirmation (CI mode)

```
$ terraform apply -auto-approve

aws_instance.web: Creating...
aws_instance.web: Creation complete after 28s

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.
```

Skips the "type yes" prompt. Use only in automation. Combine with a saved plan for safety.

### Experiment 11: Tear it all down

```
$ terraform destroy

Plan: 0 to add, 0 to change, 1 to destroy.

Do you really want to destroy all resources?
  Enter a value: yes

aws_instance.web: Destroying... [id=i-0123456789abcdef0]
aws_instance.web: Destruction complete after 41s

Destroy complete! Resources: 1 destroyed.
```

Destroy with confirmation prompt.

### Experiment 12: Destroy without confirmation

```
$ terraform destroy -auto-approve

Destroy complete! Resources: 1 destroyed.
```

For tearing down preview environments in CI.

### Experiment 13: Refresh state without changing anything

```
$ terraform apply -refresh-only

Note: Objects have changed outside of Terraform

Would you like to update the Terraform state to reflect these changes?
  Enter a value: yes

Apply complete! Resources: 0 added, 0 changed, 0 destroyed.
```

Updates state to match reality. Doesn't make any changes to cloud. Useful when you've made intentional out-of-band changes.

### Experiment 14: List everything in state

```
$ terraform state list
aws_instance.web
aws_security_group.web
aws_subnet.main
data.aws_ami.ubuntu
module.vpc.aws_vpc.main
module.vpc.aws_subnet.public[0]
module.vpc.aws_subnet.public[1]
```

Every resource Terraform manages, by address.

### Experiment 15: Show one resource's full state

```
$ terraform state show aws_instance.web

# aws_instance.web:
resource "aws_instance" "web" {
    ami                    = "ami-0c55b159cbfafe1f0"
    arn                    = "arn:aws:ec2:us-east-1:123456789012:instance/i-0123..."
    associate_public_ip_address = true
    availability_zone      = "us-east-1a"
    id                     = "i-0123456789abcdef0"
    instance_type          = "t3.micro"
    private_ip             = "10.0.1.15"
    public_ip              = "54.123.45.67"
    ...
}
```

Every attribute Terraform tracks for this resource. Useful for "what does Terraform think this looks like?"

### Experiment 16: Rename a resource without recreating it

You renamed `aws_instance.web` to `aws_instance.frontend` in your `.tf`. Plan will say "destroy web, create frontend." That's wrong — you want to keep the same instance.

```
$ terraform state mv aws_instance.web aws_instance.frontend
Successfully moved 1 object(s).
```

Now state knows that what was called `web` is now `frontend`. Next plan: no changes.

### Experiment 17: Tell Terraform to forget about a resource (without destroying)

```
$ terraform state rm aws_instance.legacy
Removed aws_instance.legacy
Successfully removed 1 resource instance(s).
```

The instance still exists in the cloud. Terraform just forgets about it. Next plan won't include it. Useful for transferring ownership to another Terraform config, or letting it be managed manually.

### Experiment 18: Adopt an existing resource

```
$ terraform import aws_instance.web i-0123456789abcdef0

aws_instance.web: Importing from ID "i-0123456789abcdef0"...
aws_instance.web: Import prepared!
aws_instance.web: Refreshing state...

Import successful!
```

The instance exists in AWS. Now Terraform also knows about it.

### Experiment 19: Show outputs

```
$ terraform output
public_ip = "54.123.45.67"
private_ip = "10.0.1.15"
```

All outputs from the root module, at once.

### Experiment 20: Get one output as JSON

```
$ terraform output -json public_ip
"54.123.45.67"

$ terraform output -raw public_ip
54.123.45.67
```

For scripting. `-raw` strips the JSON quoting.

### Experiment 21: List workspaces

```
$ terraform workspace list
* default
  staging
  prod
```

The `*` is the current workspace.

### Experiment 22: Create a new workspace

```
$ terraform workspace new feature-foo
Created and switched to workspace "feature-foo"!
```

Empty state. You're now isolated.

### Experiment 23: Switch to a different workspace

```
$ terraform workspace select prod
Switched to workspace "prod".
```

State, plan, apply all now operate against the prod state.

### Experiment 24: Visualize the dependency graph

```
$ terraform graph | dot -Tpng > graph.png
```

If you have Graphviz installed (`brew install graphviz`), this draws a picture of your dependency graph. Every resource is a node, every dependency is an edge. Useful for huge configs.

### Experiment 25: List installed providers

```
$ terraform providers

Providers required by configuration:
.
├── provider[registry.terraform.io/hashicorp/aws] ~> 5.0
└── module.vpc
    └── provider[registry.terraform.io/hashicorp/aws] >= 4.0

Providers required by state:

    provider[registry.terraform.io/hashicorp/aws]
```

Shows which providers your config and state need, and any version constraints from modules.

### Experiment 26: Lock providers for multiple platforms

```
$ terraform providers lock -platform=linux_amd64 -platform=darwin_arm64 -platform=darwin_amd64

- Fetching hashicorp/aws 5.31.0 for linux_amd64...
- Retrieved hashicorp/aws 5.31.0 for linux_amd64 (signed by HashiCorp)
- Fetching hashicorp/aws 5.31.0 for darwin_arm64...
...
```

Updates `.terraform.lock.hcl` with checksums for multiple platforms. Important when teammates use different machines than CI.

### Experiment 27: Open the interactive console

```
$ terraform console
> var.region
"us-east-1"
> length(aws_instance.web)
3
> aws_instance.web[0].public_ip
"54.123.45.67"
> upper("hello")
"HELLO"
> exit
```

Live REPL for HCL expressions. Great for testing functions and exploring state. Press Ctrl-D or type `exit` to quit.

### Experiment 28: Login to Terraform Cloud

```
$ terraform login app.terraform.io
Terraform will request an API token for app.terraform.io using your browser.
...
Token for app.terraform.io: ...

Welcome to Terraform Cloud!
```

If you use Terraform Cloud as your backend, this stores the API token in `~/.terraform.d/credentials.tfrc.json`.

### Experiment 29: Run tflint

```
$ tflint
1 issue(s) found:

Warning: Module source "git::https://github.com/foo/bar.git" is not pinned (terraform_module_pinned_source)

  on main.tf line 5:
   5:   source = "git::https://github.com/foo/bar.git"
```

`tflint` is a third-party linter. Catches common mistakes Terraform itself misses: unpinned versions, deprecated arguments, missing required tags. Install via `brew install tflint`.

### Experiment 30: Run tfsec

```
$ tfsec .

Result #1 HIGH S3 Bucket does not have logging enabled.
─────────────────────────────────────────────────────────
  main.tf:23 resource "aws_s3_bucket" "data" {
```

`tfsec` (or `checkov`) scans for security issues: open security groups, unencrypted buckets, public S3, etc. Highly recommend in CI.

### Experiment 31: Generate documentation from your modules

```
$ terraform-docs markdown table . > README.md
```

`terraform-docs` reads your `.tf` files and generates a markdown table of variables and outputs. Run this in CI to keep module READMEs accurate.

## Common Confusions

These are real questions you will eventually have. Here are real answers.

### "Should I commit terraform.tfstate to git?"

**No. Never.**

Two reasons:

1. State files often contain secrets (DB passwords, API tokens, IPs).
2. Git is not a concurrency control system; two people running apply will corrupt state.

**Use a remote backend** (S3, GCS, Azure Blob, Terraform Cloud). Add `terraform.tfstate` and `*.tfstate.backup` to your `.gitignore` from day one.

### "Should I commit `.terraform/` to git?"

**No.** That's the cached providers and modules. It's huge, platform-specific, and regeneratable. Add `.terraform/` to `.gitignore`.

### "Should I commit `.terraform.lock.hcl`?"

**Yes.** That pins the exact provider versions. Committing it ensures everyone (you, teammates, CI) uses the same versions. This is good and important.

### "Why is my plan showing destroys I didn't ask for?"

Three common reasons:

1. **Provider upgrade changed schema**: a new provider version renamed an attribute, deprecated something, or rewrote the resource definition. Terraform sees the change as a destroy+create. Read the provider's CHANGELOG.
2. **State drift**: someone changed the resource outside Terraform, and Terraform now thinks it needs to fix or recreate it.
3. **Renamed local label**: you renamed `aws_instance.web` to `aws_instance.frontend`. From Terraform's view, web disappeared and frontend appeared. Use `terraform state mv` to fix.
4. **Force-replacement attribute changed**: some attributes are immutable. Changing them requires destroying and recreating. Look for `# forces replacement` in the plan diff.

Always read the FULL plan before applying. If you see surprise destroys, stop and figure out why.

### "What's the difference between count and for_each?"

`count` indexes by integer. `for_each` indexes by string key.

Use `for_each` when:

- The set of items is named (regions, environments, users).
- You want to add or remove from the middle without recreating the whole list.
- The items have keys that mean something.

Use `count` when:

- It's a simple "make N copies of this" with no individual identity.
- The number itself is the right model (e.g., a fleet of identical workers).
- You're conditionally creating ZERO or ONE of something (`count = var.create ? 1 : 0`).

In doubt, use `for_each`.

### "What's a module?"

A module is a folder of `.tf` files used as a unit. The "root module" is the directory you ran terraform from. Any module called from there is a "child module."

Modules let you encapsulate complexity. A VPC module might internally use 30 resources but expose just 5 input variables and 5 output values. Users don't need to know the internals.

Public modules live on the **Terraform Registry** (`registry.terraform.io`). The big ones are very high quality (`terraform-aws-modules/vpc/aws`, `terraform-aws-modules/eks/aws`, `terraform-aws-modules/rds/aws`). Use them.

### "Should I use Terraform or CloudFormation/Pulumi/CDK/Crossplane?"

- **Terraform**: multi-cloud, HCL, biggest ecosystem, biggest community. Default choice in 2026.
- **CloudFormation**: AWS-only, JSON/YAML, slower, but native to AWS. Pick this if your team is AWS-only and very console-driven.
- **AWS CDK**: AWS-only, but uses real programming languages (TypeScript, Python, etc.) that compile to CloudFormation. Nice for AWS-native teams who prefer code.
- **Pulumi**: multi-cloud, real programming languages (TS, Python, Go, etc.). Same idea as CDK but multi-cloud. Trade-off: more flexibility, but also more rope to hang yourself.
- **Crossplane**: Kubernetes-native infra. You install it as a controller in k8s and define infra as k8s resources. Cool for k8s-first teams.

For most people, Terraform. See `cs iac pulumi`, `cs iac crossplane` for alternatives.

### "Why is `terraform apply` so slow?"

Most of the time is API calls. Cloud APIs are slow (sometimes seconds per call). Common speed-ups:

- **Parallelism**: by default Terraform runs 10 parallel API calls. Bump it: `terraform apply -parallelism=20`.
- **Targeted apply**: if you only need to change one thing, `terraform apply -target=aws_instance.web` skips everything else (use sparingly — easy to skip dependencies you needed).
- **Provider tuning**: some providers have aggressive timeouts; some have separate Get/Update/Create timeouts you can tune in the resource block.
- **Smaller state**: huge state files (5000+ resources) get slow. Split into multiple root modules.

If a single resource is slow, that's the cloud's API, not Terraform.

### "What does 'tainted' mean?"

A resource marked tainted will be destroyed and recreated on the next apply. Historically you used `terraform taint <addr>` to force a recreation.

In modern Terraform (1.0+), use `terraform apply -replace=<addr>` instead:

```
$ terraform apply -replace=aws_instance.web
```

Same effect, doesn't pollute state.

### "Should I use workspaces or separate root modules per environment?"

**Separate root modules.** See the workspaces section. Workspaces are for ephemeral isolation, not for separating prod from dev.

### "What's a backend lock and why does it exist?"

A backend lock prevents two people from running apply at the same time. Without it, two parallel applies could each write a different state, leading to corruption.

The lock is acquired before any state-mutating operation (apply, destroy, state mv, etc.). It's released when the operation completes. If a process crashes, the lock can become stale — use `terraform force-unlock` to clear it.

### "How do I roll back a Terraform apply?"

**There is no rollback button.** Terraform doesn't have one.

The closest equivalents:

1. **Re-apply the previous version of your `.tf`**. Check out the previous git commit, run apply. Terraform will compute a new plan that reverses your last change.
2. **State file versioning**. If you have S3 versioning on, you can fetch a previous state file, but using it means accepting that reality has moved on while state thinks it hasn't. Risky.
3. **For TF Cloud**: there's a "Discard run" button and a state file history. You can roll back to a previous state version.

The clean answer: use git to roll back your `.tf`, then re-apply. Treat your `.tf` files as the source of truth, and use git's rollback machinery.

### "Why doesn't terraform plan show the same diff I see in apply?"

It should — `apply` runs plan again unless you pass a saved plan file. If they differ, it's because reality changed between your plan and your apply. Use saved plan files (`plan -out=plan.bin && apply plan.bin`) to lock in the diff.

### "What's the difference between Terraform and Ansible?"

- **Terraform**: creates infrastructure (VMs, networks, databases). Works through cloud APIs. Owns and tracks lifecycle.
- **Ansible**: configures the inside of those VMs (install packages, lay down config files, deploy apps). Works through SSH or local execution.

You use both. Terraform creates the EC2 instance. Ansible installs nginx on it. See `cs ramp-up ansible-eli5`.

### "What's HCL vs. Terraform vs. HashiCorp?"

- **HashiCorp**: a company.
- **Terraform**: a tool made by HashiCorp.
- **HCL** (HashiCorp Configuration Language): the syntax used by Terraform's `.tf` files. Also used by other HashiCorp tools (Nomad, Consul, Vault, Packer).

HCL is just a language. Terraform is the engine that reads it.

### "Can I use JSON instead of HCL?"

Yes. Terraform accepts `.tf.json` files with the same structure expressed in JSON. Useful for machine-generated configurations. For humans, HCL is much nicer.

### "Why does my apply hang on a destroy?"

The cloud might be slow at deleting that resource, OR the resource has dependencies (an EBS volume attached, a network interface, a public IP) that haven't been cleaned up. AWS especially can take 5-10 minutes to fully delete some resources.

Sometimes it's a circular issue (a security group references itself or another that references back). Check the cloud console to see what's happening on that resource.

## Vocabulary

| Term | Plain English |
|------|---------------|
| **Terraform** | The tool. A CLI that reads .tf files and calls cloud APIs to make reality match. |
| **HCL** | HashiCorp Configuration Language. The syntax used in .tf files. |
| **HashiCorp** | The company that makes Terraform (also Vault, Consul, Nomad, etc.). |
| **`.tf` file** | A text file written in HCL. The blueprint. |
| **`.tf.json` file** | A .tf file written in JSON instead of HCL. Same meaning. |
| **`.tfvars` file** | A file holding variable values. Loaded automatically if named terraform.tfvars or *.auto.tfvars. |
| **`.terraform/` folder** | Cache for downloaded providers and modules. Don't commit. |
| **`.terraform.lock.hcl`** | Lockfile pinning exact provider versions and checksums. DO commit. |
| **provider** | A plugin that knows how to talk to one specific service (AWS, GCP, GitHub, etc.). |
| **plugin** | Same as provider; the binary form. |
| **registry** | A place to download providers and modules. The default is registry.terraform.io. |
| **source** | Where to get a provider or module from. E.g., `hashicorp/aws`. |
| **version** | Which version of a provider or module to use. |
| **version constraint** | A rule for which versions are acceptable. E.g., `~> 5.0`, `>= 4.0`, `!= 5.1.0`. |
| **`~>`** | Pessimistic version constraint. `~> 5.0` means "any 5.x but not 6.x." |
| **`required_providers`** | Block listing which providers your config needs. |
| **`required_version`** | Constraint on which Terraform CLI version is acceptable. |
| **terraform block** | The meta-block. Contains `required_providers`, `backend`, `required_version`, etc. |
| **backend** | Where Terraform stores the state file. Local, S3, GCS, Azure, TFC, Consul, HTTP. |
| **local backend** | State stored on the local filesystem. The default. Fine for solo work. |
| **S3 backend** | State stored in an S3 bucket. Use DynamoDB for locking. Most common for AWS. |
| **GCS backend** | State stored in Google Cloud Storage. Native locking. |
| **Azure backend** | State stored in Azure Blob Storage. Lease-based locking. |
| **Terraform Cloud** | HashiCorp's managed service. Free tier exists. UI, RBAC, state, runs, all integrated. |
| **remote state** | Any state stored not on your local disk. Required for teamwork. |
| **state file** | The JSON record of every resource Terraform manages. The brain. |
| **`terraform.tfstate`** | The default name of the local state file. |
| **`terraform.tfstate.backup`** | The previous version of the state file. |
| **state locking** | Ensures only one apply runs at a time. Prevents corruption. |
| **DynamoDB lock** | An AWS DynamoDB table used to lock S3-backed state. |
| **force-unlock** | Manually release a stuck state lock. Use with extreme caution. |
| **resource** | A block declaring something Terraform should create and manage. |
| **data source** | A block declaring something Terraform should READ (read-only). |
| **module** | A folder of .tf files used as a unit. Reusable building block. |
| **root module** | The directory you ran `terraform` from. The top-level module. |
| **child module** | Any module called from another module. |
| **module source** | Where to get a module from: registry, git, local path, http, s3. |
| **variable** | An input to your config. Declared with `variable "name" {}`. Used as `var.name`. |
| **output** | A value exposed by your config. Declared with `output "name" {}`. |
| **local** | A computed value defined in a `locals` block. Used as `local.name` (singular). |
| **expression** | A computed value: math, function calls, references, conditionals. |
| **for_each** | Make N copies of a resource, keyed by a map or set. Stable across changes. |
| **count** | Make N copies of a resource, indexed by integer. Fragile in the middle. |
| **dynamic block** | A way to repeat a sub-block (like `ingress`) inside a resource. |
| **`depends_on`** | Explicit dependency declaration. Use when implicit deps don't work. |
| **implicit dependency** | A dependency Terraform infers from references between resources. |
| **explicit dependency** | A dependency declared with `depends_on`. |
| **provisioner** | A way to run a command after a resource is created. Avoid when possible. |
| **`local-exec`** | A provisioner that runs on the machine running Terraform. |
| **`remote-exec`** | A provisioner that SSHs into the resource and runs commands. |
| **connection** | A block configuring how `remote-exec` connects (SSH user, key, host). |
| **lifecycle** | A block inside a resource controlling its lifecycle behavior. |
| **`create_before_destroy`** | Lifecycle setting: create the new one before destroying the old one. |
| **`prevent_destroy`** | Lifecycle setting: refuse to destroy this resource. |
| **`ignore_changes`** | Lifecycle setting: ignore changes to specific attributes. |
| **`replace_triggered_by`** | Lifecycle setting: replace this resource when another resource changes. |
| **taint** | Mark a resource for replacement on next apply. Use `apply -replace` instead in modern TF. |
| **untaint** | Undo a taint. Rarely needed in modern Terraform. |
| **import** | Bring an existing cloud resource into Terraform state. |
| **refresh** | Update state from cloud reality. `apply -refresh-only` is the modern form. |
| **plan** | Compute the diff between desired state and current state. |
| **apply** | Execute the plan against the cloud. |
| **destroy** | Tear down everything in state. |
| **validate** | Check that .tf files parse and references resolve. |
| **fmt** | Format .tf files to canonical layout. |
| **init** | Download providers, set up backend. The first command in any project. |
| **console** | Interactive REPL for HCL expressions. |
| **graph** | Generate a dependency graph in DOT format. |
| **workspace** | A logical name for a separate state. Inside one config. |
| **default workspace** | The workspace created automatically. Where you start. |
| **named workspace** | Any workspace you create with `terraform workspace new`. |
| **`terraform.workspace`** | Built-in expression for the current workspace name. |
| **TF_VAR_*** | Environment variables that set Terraform variables. `TF_VAR_region` sets `var.region`. |
| **variable precedence** | The order Terraform picks variable values: -var > -var-file > .auto.tfvars > .tfvars > env > default. |
| **sensitive (variable)** | A variable whose value won't be printed in logs. |
| **sensitive (output)** | An output whose value won't be printed by apply (must use `terraform output` to fetch). |
| **nullable** | Variable can be null. Default is true. |
| **optional** | An object attribute that can be omitted. |
| **type constraint** | string, number, bool, list, set, map, tuple, object, any. |
| **validation block** | A block inside `variable` declaring custom rules. |
| **moved block** | A block declaring a resource was renamed. Tells Terraform to update state, not destroy/recreate. |
| **removed block** | A block declaring a resource was removed but should NOT be destroyed (just removed from state). TF 1.7+. |
| **check block** | A block (TF 1.5+) declaring an expectation. Like an assertion that runs at plan time. |
| **precondition** | An assertion that must hold before a resource is created. |
| **postcondition** | An assertion that must hold after a resource is created. |
| **terraform-docs** | Tool to auto-generate module documentation from .tf files. |
| **tflint** | Linter for .tf files. Catches stylistic and best-practice issues. |
| **tfsec** | Security scanner for .tf files. Finds insecure configs. |
| **checkov** | Another security scanner; broader coverage than tfsec. |
| **terraform-compliance** | BDD-style policy testing for Terraform plans. |
| **Terragrunt** | A wrapper around Terraform for keeping configs DRY across environments. |
| **Atlantis** | A self-hosted PR-bot for Terraform. Plans on PR, applies on comment. |
| **Spacelift** | Hosted Terraform CI/CD platform. Atlantis-as-a-service alternative. |
| **env0** | Hosted Terraform CI/CD platform. |
| **Scalr** | Hosted Terraform platform. |
| **drift** | When real cloud state differs from what state file says. |
| **DAG** | Directed Acyclic Graph. Terraform's internal dependency model. |
| **plan file** | A binary file from `terraform plan -out=`. Reusable in `terraform apply`. |

## Try This

Some safe experiments. Start in a throwaway directory. Use a free-tier-eligible AWS account.

### Experiment 1: Spin up a t3.micro the lazy way

Use the official `terraform-aws-modules/ec2-instance/aws` module. Save this as `main.tf`:

```hcl
terraform {
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 5.0" }
  }
}

provider "aws" {
  region = "us-east-1"
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"]
  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }
}

module "ec2_instance" {
  source  = "terraform-aws-modules/ec2-instance/aws"
  version = "~> 5.0"

  name = "tf-eli5-test"

  instance_type = "t3.micro"
  ami           = data.aws_ami.ubuntu.id

  tags = {
    Project = "tf-eli5"
  }
}

output "instance_id" {
  value = module.ec2_instance.id
}

output "public_ip" {
  value = module.ec2_instance.public_ip
}
```

Then:

```
$ terraform init
$ terraform plan
$ terraform apply
```

When done playing, **don't forget**:

```
$ terraform destroy
```

Or you'll keep paying for the instance.

### Experiment 2: Manually drift the resource

After your apply succeeds, go to the AWS console. Find the instance. Add a new tag like `ManuallyChanged = true` directly in the console.

Now back in your terminal:

```
$ terraform plan
```

See the drift? Terraform notices the new tag and offers to remove it (because your `.tf` doesn't say it should be there). Either accept the apply (which removes the tag), or update your `.tf` to include it.

### Experiment 3: Import an existing resource

Manually create an EC2 instance in the console. Note its ID (`i-xxx`).

In a fresh `.tf`:

```hcl
resource "aws_instance" "imported" {
  # leave this empty for now
}
```

Then:

```
$ terraform import aws_instance.imported i-xxx
$ terraform state show aws_instance.imported
```

Now copy the attributes shown into your `.tf`. Run `plan` until there are no changes. Now Terraform manages this resource.

### Experiment 4: Try refresh-only

After your apply, change something via the console. Then:

```
$ terraform apply -refresh-only
```

State now matches reality (without changing reality). Subsequent `plan` will show your `.tf` is "out of date" if it disagreed.

### Experiment 5: Save a plan and apply it later

```
$ terraform plan -out=p1.bin
$ # ... go to lunch, come back ...
$ terraform apply p1.bin
```

The plan was locked in. Even if state changed in the meantime, the plan will still try to apply what it captured. You'll see warnings if anything has shifted.

### Experiment 6: Break it on purpose

Change a hardcoded `instance_type = "t3.micro"` to `"t3.tiny"` (which doesn't exist). Run plan. Watch the validation fail.

Add a typo: `instnace_type`. Run plan. Watch the unsupported argument error.

Make a cycle: `aws_security_group.a` has rule referencing `aws_security_group.b.id`, and `aws_security_group.b` has rule referencing `aws_security_group.a.id`. Run plan. Watch the cycle error.

Each error teaches you to read Terraform's diagnostics.

### Experiment 7: Try terraform console

```
$ terraform console
> 1 + 2
3
> upper("hello world")
"HELLO WORLD"
> length(["a", "b", "c"])
3
> [for n in ["a","b","c"] : upper(n)]
[
  "A",
  "B",
  "C",
]
> jsonencode({a = 1, b = "two"})
"{\"a\":1,\"b\":\"two\"}"
> exit
```

Try a bunch of expressions. The console is the fastest way to learn HCL syntax.

### Experiment 8: Visualize your graph

If you have several resources that depend on each other, install Graphviz and run:

```
$ brew install graphviz
$ terraform graph | dot -Tpng > graph.png
$ open graph.png
```

You'll see every resource as a node and every dependency as an arrow. Big configs become much easier to reason about.

### Experiment 9: Try a workspace

```
$ terraform workspace new feature-foo
$ terraform plan
```

Notice plan says "0 resources" — you're in a fresh state. Apply something. Switch back to default. Notice the original resources are still there.

```
$ terraform workspace select feature-foo
$ terraform destroy
$ terraform workspace select default
$ terraform workspace delete feature-foo
```

### Experiment 10: Run tflint and tfsec

```
$ brew install tflint tfsec
$ tflint
$ tfsec .
```

See what each tool flags. Even on a small config, tfsec will find things to nag you about (like unencrypted EBS volumes).

## Where to Go Next

Once this sheet feels easy, the dense engineer-grade material is one command away. Stay in the terminal:

- **`cs config-mgmt terraform`** — the dense reference for the Terraform CLI. Every flag, every subcommand, every nook.
- **`cs detail config-mgmt/terraform`** — academic underpinning: graph algorithms, diff math, the formal model behind plan and apply.
- **`cs iac terragrunt`** — Terragrunt is a thin wrapper around Terraform that helps with DRY and per-environment patterns.
- **`cs iac pulumi`** — Pulumi: same idea as Terraform but with real programming languages (TypeScript, Python, Go, etc.) instead of HCL.
- **`cs iac crossplane`** — Crossplane: Kubernetes-native infrastructure as code.
- **`cs config-mgmt ansible`** — Ansible handles the inside of the VMs Terraform creates. Together they're the standard combo.
- **`cs ramp-up ansible-eli5`** — partner ELI5 sheet for Ansible.
- **`cs ramp-up kubernetes-eli5`** — Terraform often creates the EKS/GKE/AKS cluster that Kubernetes runs in.
- **`cs ramp-up linux-kernel-eli5`** — what's actually running on the boxes Terraform created.
- **`cs config-mgmt puppet`**, **`cs config-mgmt chef`**, **`cs config-mgmt salt`** — alternative configuration-management tools.
- **`cs cloud aws-cli`**, **`cs cloud gcloud`**, **`cs cloud azure-cli`** — the lower-level CLIs Terraform calls under the hood.

## See Also

- `config-mgmt/terraform` — engineer-grade Terraform reference.
- `config-mgmt/ansible` — config management for the inside of the VMs.
- `config-mgmt/puppet` — alternative config management tool.
- `config-mgmt/chef` — alternative config management tool.
- `config-mgmt/salt` — alternative config management tool.
- `iac/terragrunt` — Terraform wrapper for keeping configs DRY.
- `iac/pulumi` — Terraform alternative using real programming languages.
- `iac/crossplane` — Kubernetes-native infra as code.
- `cloud/aws-cli` — the AWS CLI Terraform calls under the hood.
- `cloud/gcloud` — the Google Cloud CLI.
- `cloud/azure-cli` — the Azure CLI.
- `ramp-up/ansible-eli5` — partner ELI5 sheet for Ansible.
- `ramp-up/kubernetes-eli5` — partner ELI5 sheet for Kubernetes.
- `ramp-up/linux-kernel-eli5` — what's running on the VMs Terraform creates.
- `ramp-up/tcp-eli5` — the network protocol your cloud resources use to talk.

## References

- **developer.hashicorp.com/terraform** — the official Terraform docs. Bookmark this.
- **registry.terraform.io** — the public registry of providers and modules.
- **"Terraform: Up & Running"** by Yevgeniy Brikman — the standard book for learning Terraform deeply.
- **"Infrastructure as Code"** by Kief Morris — broader IaC theory and practice.
- **terraform-best-practices.com** — community-maintained best-practice document.
- **HCL spec** at hcl-lang.dev — formal HCL language specification.
- **Terraform Provider Development docs** — for writing your own provider.
- **terraform-docs.io** — auto-generate module documentation.
- **github.com/terraform-linters/tflint** — linter for Terraform.
- **aquasecurity.github.io/tfsec** — security scanner.
- **www.checkov.io** — broader policy-as-code scanner for Terraform.

— End of ELI5 —

When this sheet feels boring (and it will, faster than you think), graduate to `cs config-mgmt terraform` — the engineer-grade reference. It uses real names for everything: every CLI flag, every meta-argument, every backend option. After that, `cs detail config-mgmt/terraform` gives you the academic underpinning: how the dependency graph is built, how the diff is computed, the math behind state versioning. By the time you've read both, you will be reading Terraform internals without a flinch.
