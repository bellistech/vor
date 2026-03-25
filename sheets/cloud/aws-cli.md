# AWS CLI v2 (Amazon Web Services Command Line Interface)

Unified tool to manage AWS services from the terminal with profiles, SSO, and JMESPath queries.

## Configuration

### Install and verify

```bash
# Check version
aws --version

# Install on macOS
curl "https://awscli.amazonaws.com/AWSCLIV2.pkg" -o "AWSCLIV2.pkg"
sudo installer -pkg AWSCLIV2.pkg -target /
```

### Configure profiles

```bash
# Interactive setup (creates ~/.aws/credentials and ~/.aws/config)
aws configure

# Configure a named profile
aws configure --profile staging

# Set individual values
aws configure set region us-west-2 --profile prod
aws configure set output json

# List configured profiles
aws configure list-profiles

# View current configuration
aws configure list
```

### SSO configuration

```bash
# Configure SSO login
aws configure sso
# Follow prompts: SSO start URL, region, account, role

# Log in via SSO
aws sso login --profile my-sso-profile

# Use SSO profile for commands
aws s3 ls --profile my-sso-profile
```

### Environment variables

```bash
# Override profile/region per-command or export
export AWS_PROFILE=staging
export AWS_REGION=eu-west-1
export AWS_DEFAULT_OUTPUT=json

# Use access keys directly (not recommended for long-term)
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=wJalr...
```

## S3

### Bucket operations

```bash
# List all buckets
aws s3 ls

# List objects in a bucket
aws s3 ls s3://my-bucket/
aws s3 ls s3://my-bucket/prefix/ --recursive

# Create a bucket
aws s3 mb s3://my-new-bucket --region us-east-1

# Delete an empty bucket
aws s3 rb s3://my-bucket

# Delete a bucket and ALL contents (dangerous)
aws s3 rb s3://my-bucket --force
```

### Copy and sync

```bash
# Copy file to S3
aws s3 cp myfile.txt s3://my-bucket/

# Copy from S3 to local
aws s3 cp s3://my-bucket/myfile.txt ./

# Copy entire directory recursively
aws s3 cp ./local-dir s3://my-bucket/prefix/ --recursive

# Sync local directory to S3 (only uploads changes)
aws s3 sync ./local-dir s3://my-bucket/prefix/

# Sync with delete (removes files in dest not in source)
aws s3 sync ./local-dir s3://my-bucket/prefix/ --delete

# Exclude/include patterns
aws s3 sync . s3://my-bucket/ --exclude "*.log" --include "important.log"
```

### Presigned URLs

```bash
# Generate a presigned URL (default 1 hour)
aws s3 presign s3://my-bucket/myfile.txt

# Custom expiration (seconds)
aws s3 presign s3://my-bucket/myfile.txt --expires-in 3600
```

## EC2

### Instances

```bash
# List all instances with key details
aws ec2 describe-instances \
  --query 'Reservations[].Instances[].{ID:InstanceId,Type:InstanceType,State:State.Name,IP:PublicIpAddress,Name:Tags[?Key==`Name`]|[0].Value}' \
  --output table

# Filter running instances
aws ec2 describe-instances \
  --filters "Name=instance-state-name,Values=running"

# Launch an instance
aws ec2 run-instances \
  --image-id ami-0abcdef1234567890 \
  --instance-type t3.micro \
  --key-name my-key \
  --security-group-ids sg-0123456789abcdef0 \
  --subnet-id subnet-0123456789abcdef0 \
  --count 1 \
  --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=my-server}]'

# Start / stop / terminate
aws ec2 start-instances --instance-ids i-0123456789abcdef0
aws ec2 stop-instances --instance-ids i-0123456789abcdef0
aws ec2 terminate-instances --instance-ids i-0123456789abcdef0
```

### Security groups

```bash
# List security groups
aws ec2 describe-security-groups --output table

# Create a security group
aws ec2 create-security-group \
  --group-name my-sg \
  --description "My security group" \
  --vpc-id vpc-0123456789abcdef0

# Add inbound rule (allow SSH)
aws ec2 authorize-security-group-ingress \
  --group-id sg-0123456789abcdef0 \
  --protocol tcp --port 22 --cidr 0.0.0.0/0

# Revoke inbound rule
aws ec2 revoke-security-group-ingress \
  --group-id sg-0123456789abcdef0 \
  --protocol tcp --port 22 --cidr 0.0.0.0/0
```

### Key pairs

```bash
# List key pairs
aws ec2 describe-key-pairs

# Create and save a key pair
aws ec2 create-key-pair --key-name my-key \
  --query 'KeyMaterial' --output text > my-key.pem
chmod 400 my-key.pem
```

## IAM

### Users and roles

```bash
# List IAM users
aws iam list-users --output table

# Create a user
aws iam create-user --user-name deploy-bot

# Create access keys for a user
aws iam create-access-key --user-name deploy-bot

# List roles
aws iam list-roles --query 'Roles[].{Name:RoleName,Arn:Arn}' --output table

# Attach a managed policy to a user
aws iam attach-user-policy \
  --user-name deploy-bot \
  --policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess

# List attached policies for a user
aws iam list-attached-user-policies --user-name deploy-bot
```

## STS

### Identity and role assumption

```bash
# Who am I?
aws sts get-caller-identity

# Assume a role (returns temporary credentials)
aws sts assume-role \
  --role-arn arn:aws:iam::123456789012:role/my-role \
  --role-session-name my-session

# Decode an authorization error message
aws sts decode-authorization-message --encoded-message <encoded-msg>
```

## Lambda

### Function management

```bash
# List functions
aws lambda list-functions --query 'Functions[].FunctionName'

# Invoke a function
aws lambda invoke --function-name my-func --payload '{"key":"value"}' output.json

# Update function code from zip
aws lambda update-function-code \
  --function-name my-func \
  --zip-file fileb://function.zip

# View function configuration
aws lambda get-function-configuration --function-name my-func

# View recent logs
aws logs tail /aws/lambda/my-func --follow
```

## CloudFormation

### Stack operations

```bash
# Deploy / update a stack
aws cloudformation deploy \
  --template-file template.yaml \
  --stack-name my-stack \
  --capabilities CAPABILITY_IAM \
  --parameter-overrides Env=prod

# List stacks
aws cloudformation list-stacks \
  --stack-status-filter CREATE_COMPLETE UPDATE_COMPLETE

# Describe stack events (for debugging)
aws cloudformation describe-stack-events --stack-name my-stack

# Delete a stack
aws cloudformation delete-stack --stack-name my-stack
```

## Route53

### DNS management

```bash
# List hosted zones
aws route53 list-hosted-zones

# List records in a zone
aws route53 list-resource-record-sets --hosted-zone-id Z0123456789

# Create/update a record (via change batch JSON)
aws route53 change-resource-record-sets \
  --hosted-zone-id Z0123456789 \
  --change-batch file://dns-change.json
```

## RDS

### Database instances

```bash
# List DB instances
aws rds describe-db-instances \
  --query 'DBInstances[].{ID:DBInstanceIdentifier,Engine:Engine,Status:DBInstanceStatus}' \
  --output table

# Create a snapshot
aws rds create-db-snapshot \
  --db-instance-identifier my-db \
  --db-snapshot-identifier my-db-snap-$(date +%Y%m%d)

# Reboot an instance
aws rds reboot-db-instance --db-instance-identifier my-db
```

## Common Patterns

### JMESPath queries and filters

```bash
# Use --query for JMESPath (client-side filtering/projection)
aws ec2 describe-instances \
  --query 'Reservations[].Instances[?State.Name==`running`].InstanceId' \
  --output text

# Use --filters for server-side filtering (faster)
aws ec2 describe-instances \
  --filters "Name=tag:Environment,Values=production"

# Combine both
aws ec2 describe-instances \
  --filters "Name=instance-state-name,Values=running" \
  --query 'Reservations[].Instances[].{ID:InstanceId,AZ:Placement.AvailabilityZone}'

# Output formats: json (default), text, table, yaml
aws ec2 describe-instances --output table
```

## Tips

- Use `--dry-run` on EC2 commands to check permissions without executing.
- Use `--no-paginate` or `--max-items` to control output size.
- Pipe JSON output through `jq` for ad-hoc filtering beyond JMESPath.
- Use `aws configure list` to debug which credentials/region are active.
- Tag everything with `--tag-specifications` for cost tracking and organization.
- Set `AWS_PAGER=""` to disable the default pager for scripting.

## References

- [AWS CLI v2 Command Reference](https://docs.aws.amazon.com/cli/latest/reference/)
- [AWS CLI User Guide](https://docs.aws.amazon.com/cli/latest/userguide/)
- [AWS CLI Configuration and Credential Files](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)
- [AWS CLI SSO Configuration](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sso.html)
- [AWS CLI Environment Variables](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html)
- [AWS CLI Output Formatting](https://docs.aws.amazon.com/cli/latest/userguide/cli-usage-output-format.html)
- [AWS CLI S3 Commands](https://docs.aws.amazon.com/cli/latest/reference/s3/)
- [JMESPath Specification](https://jmespath.org/specification.html)
- [JMESPath Tutorial](https://jmespath.org/tutorial.html)
- [AWS CLI GitHub Repository](https://github.com/aws/aws-cli)
- [AWS CLI v2 Install Guide](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
