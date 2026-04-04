# pulumi (infrastructure as code)

Pulumi is an infrastructure-as-code platform that uses general-purpose programming languages (Go, Python, TypeScript, C#, Java) to define cloud resources, supporting state management, secrets encryption, policy enforcement, and an automation API for programmatic infrastructure workflows.

## Project Setup

### Initialize and Configure

```bash
# Install Pulumi CLI
curl -fsSL https://get.pulumi.com | sh

# Create new project (interactive)
pulumi new go
pulumi new python
pulumi new typescript
pulumi new aws-go
pulumi new kubernetes-python

# Create project in existing directory
pulumi new go --dir ./infra --name my-project

# Login to backend
pulumi login                          # Pulumi Cloud (default)
pulumi login --local                  # Local filesystem
pulumi login s3://my-state-bucket     # S3 backend
pulumi login gs://my-state-bucket     # GCS backend
pulumi login azblob://my-container    # Azure Blob

# Set config values
pulumi config set aws:region us-east-1
pulumi config set dbName myapp

# Set secret config values (encrypted)
pulumi config set --secret dbPassword hunter2

# View config
pulumi config

# Stack management
pulumi stack init dev
pulumi stack init staging
pulumi stack init prod
pulumi stack ls
pulumi stack select dev
pulumi stack rm old-stack
```

## Deployment Operations

### Preview and Deploy

```bash
# Preview changes (dry run)
pulumi preview

# Preview with detailed diff
pulumi preview --diff

# Deploy infrastructure
pulumi up

# Deploy with auto-approve
pulumi up --yes

# Deploy specific target
pulumi up --target urn:pulumi:dev::myproject::aws:s3/bucket:Bucket::my-bucket

# Destroy all resources
pulumi destroy

# Destroy with auto-approve
pulumi destroy --yes

# Refresh state from cloud
pulumi refresh

# Export stack state
pulumi stack export > state.json

# Import stack state
pulumi stack import < state.json

# Import existing resource
pulumi import aws:s3/bucket:Bucket my-bucket my-existing-bucket

# Cancel in-progress update
pulumi cancel

# Stack outputs
pulumi stack output
pulumi stack output bucketName
pulumi stack output --json
```

## Go Examples

### Resources and Components

```go
package main

import (
    "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/s3"
    "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
    pulumi.Run(func(ctx *pulumi.Context) error {
        // Read config
        cfg := config.New(ctx, "")
        env := cfg.Require("environment")

        // Create S3 bucket
        bucket, err := s3.NewBucket(ctx, "data-bucket", &s3.BucketArgs{
            Bucket: pulumi.Sprintf("myapp-%s-data", env),
            Tags: pulumi.StringMap{
                "Environment": pulumi.String(env),
            },
        })
        if err != nil {
            return err
        }

        // Create VPC
        vpc, err := ec2.NewVpc(ctx, "main-vpc", &ec2.VpcArgs{
            CidrBlock:          pulumi.String("10.0.0.0/16"),
            EnableDnsHostnames: pulumi.Bool(true),
            EnableDnsSupport:   pulumi.Bool(true),
        })
        if err != nil {
            return err
        }

        // Export outputs
        ctx.Export("bucketArn", bucket.Arn)
        ctx.Export("vpcId", vpc.ID())
        return nil
    })
}
```

### ComponentResource Pattern

```go
// Reusable component (e.g., VPC with subnets)
type VpcComponent struct {
    pulumi.ResourceState
    VpcId    pulumi.IDOutput
    SubnetIds pulumi.StringArrayOutput
}

func NewVpcComponent(ctx *pulumi.Context, name string,
    cidr string, azs []string, opts ...pulumi.ResourceOption) (*VpcComponent, error) {

    component := &VpcComponent{}
    err := ctx.RegisterComponentResource("custom:network:Vpc", name, component, opts...)
    if err != nil {
        return nil, err
    }

    vpc, err := ec2.NewVpc(ctx, name+"-vpc", &ec2.VpcArgs{
        CidrBlock: pulumi.String(cidr),
    }, pulumi.Parent(component))
    if err != nil {
        return nil, err
    }

    component.VpcId = vpc.ID()
    return component, nil
}
```

## Python Examples

### Resources and Stacks

```python
import pulumi
import pulumi_aws as aws

config = pulumi.Config()
env = config.require("environment")

# Create S3 bucket
bucket = aws.s3.Bucket("data-bucket",
    bucket=f"myapp-{env}-data",
    tags={"Environment": env},
)

# Create EC2 instance
instance = aws.ec2.Instance("web-server",
    ami="ami-0c55b159cbfafe1f0",
    instance_type="t3.micro",
    tags={"Name": f"web-{env}"},
)

# Stack references (cross-stack)
network_stack = pulumi.StackReference("org/network/dev")
vpc_id = network_stack.get_output("vpcId")

# Exports
pulumi.export("bucket_name", bucket.id)
pulumi.export("instance_ip", instance.public_ip)
```

## State Management

### Backend Configuration

```bash
# View current backend
pulumi whoami -v

# Migrate state between backends
pulumi stack export --stack dev > state.json
pulumi login s3://new-backend
pulumi stack init dev
pulumi stack import < state.json

# State file encryption
pulumi stack change-secrets-provider \
  "awskms://alias/pulumi-secrets?region=us-east-1"

# Lock state (Pulumi Cloud)
# Automatic -- concurrent updates are blocked

# Checkpoints and history
pulumi stack history
pulumi stack history --show-secrets
```

## Automation API

### Programmatic Infrastructure

```go
package main

import (
    "context"
    "fmt"
    "github.com/pulumi/pulumi/sdk/v3/go/auto"
    "github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
)

func main() {
    ctx := context.Background()

    // Create or select stack
    stack, err := auto.UpsertStackInlineSource(ctx, "dev", "myproject",
        func(ctx *pulumi.Context) error {
            // Define resources inline
            return nil
        })
    if err != nil {
        panic(err)
    }

    // Set config
    stack.SetConfig(ctx, "aws:region",
        auto.ConfigValue{Value: "us-east-1"})

    // Preview
    preview, _ := stack.Preview(ctx)
    fmt.Println(preview.StdOut)

    // Deploy
    result, _ := stack.Up(ctx, optup.ProgressStreams(os.Stdout))
    fmt.Println("outputs:", result.Outputs)

    // Destroy
    stack.Destroy(ctx)
}
```

## Policy as Code (CrossGuard)

### Policy Packs

```typescript
// policy/index.ts
import * as policy from "@pulumi/policy";

new policy.PolicyPack("security-policies", {
    policies: [
        {
            name: "no-public-s3",
            description: "S3 buckets must not be public",
            enforcementLevel: "mandatory",
            validateResource: policy.validateResourceOfType(
                aws.s3.Bucket, (bucket, args, reportViolation) => {
                    if (bucket.acl === "public-read") {
                        reportViolation("S3 buckets must not be public");
                    }
                }),
        },
        {
            name: "required-tags",
            description: "All resources must have required tags",
            enforcementLevel: "mandatory",
            validateStack: (args, reportViolation) => {
                for (const r of args.resources) {
                    if (r.props.tags && !r.props.tags["Environment"]) {
                        reportViolation(`${r.name} missing Environment tag`);
                    }
                }
            },
        },
    ],
});
```

```bash
# Run with policy pack
pulumi up --policy-pack ./policy

# Publish policy pack to Pulumi Cloud
pulumi policy publish ./policy

# Enable policy pack for org
pulumi policy enable org/security-policies latest
```

## Tips

- Use `pulumi preview --diff` before every deployment to see exact property-level changes
- Store secrets with `pulumi config set --secret` to encrypt them in state rather than plaintext
- Use `ComponentResource` to create reusable abstractions that encapsulate multiple resources
- Stack references enable cross-stack outputs for network/compute separation without hardcoding
- The Automation API embeds Pulumi in applications for self-service infrastructure or CI/CD pipelines
- Use `--target` to deploy individual resources during debugging without touching the full stack
- S3/GCS/Azure Blob backends give you state management without Pulumi Cloud dependency
- Always export essential outputs (IDs, ARNs, endpoints) for downstream consumption
- Policy packs with `mandatory` enforcement prevent non-compliant resources from being created
- Run `pulumi refresh` after manual cloud console changes to sync state before the next deployment

## See Also

- terraform, terragrunt, crossplane, cloudformation, cdk

## References

- [Pulumi Documentation](https://www.pulumi.com/docs/)
- [Pulumi Registry (Providers)](https://www.pulumi.com/registry/)
- [Pulumi Examples](https://github.com/pulumi/examples)
- [Automation API Guide](https://www.pulumi.com/docs/using-pulumi/automation-api/)
- [CrossGuard Policy as Code](https://www.pulumi.com/docs/using-pulumi/crossguard/)
