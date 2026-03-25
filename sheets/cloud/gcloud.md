# gcloud (Google Cloud CLI)

Command-line tool for managing Google Cloud Platform resources, services, and configurations.

## Authentication

### Login and credentials

```bash
# Interactive browser login
gcloud auth login

# Application default credentials (for local dev with client libraries)
gcloud auth application-default login

# Authenticate with a service account key file
gcloud auth activate-service-account \
  --key-file=service-account.json

# List authenticated accounts
gcloud auth list

# Revoke credentials
gcloud auth revoke user@example.com

# Print access token (useful for curl/API calls)
gcloud auth print-access-token
```

## Configuration

### Projects and properties

```bash
# Set the active project
gcloud config set project my-project-id

# Set default region and zone
gcloud config set compute/region us-central1
gcloud config set compute/zone us-central1-a

# View current configuration
gcloud config list

# Unset a property
gcloud config unset compute/zone
```

### Named configurations

```bash
# Create a named configuration (e.g., per-project or per-env)
gcloud config configurations create staging

# Switch between configurations
gcloud config configurations activate staging

# List configurations
gcloud config configurations list

# Override project for a single command
gcloud compute instances list --project other-project
```

## Compute Engine

### Instances

```bash
# List all instances
gcloud compute instances list

# Create an instance
gcloud compute instances create my-vm \
  --machine-type=e2-medium \
  --image-family=debian-12 \
  --image-project=debian-cloud \
  --boot-disk-size=20GB \
  --tags=http-server

# SSH into an instance
gcloud compute ssh my-vm --zone us-central1-a

# Start / stop / delete
gcloud compute instances start my-vm
gcloud compute instances stop my-vm
gcloud compute instances delete my-vm

# Describe an instance
gcloud compute instances describe my-vm --zone us-central1-a
```

### Firewall rules

```bash
# List firewall rules
gcloud compute firewall-rules list

# Allow HTTP traffic
gcloud compute firewall-rules create allow-http \
  --allow tcp:80 \
  --target-tags http-server \
  --source-ranges 0.0.0.0/0

# Delete a rule
gcloud compute firewall-rules delete allow-http
```

### Disks

```bash
# List disks
gcloud compute disks list

# Create a disk
gcloud compute disks create my-disk --size=50GB --type=pd-ssd

# Attach a disk to an instance
gcloud compute instances attach-disk my-vm --disk my-disk

# Create a snapshot
gcloud compute disks snapshot my-disk --snapshot-names my-snap
```

## GKE (Google Kubernetes Engine)

### Cluster management

```bash
# Create a cluster
gcloud container clusters create my-cluster \
  --num-nodes=3 \
  --machine-type=e2-standard-4 \
  --region us-central1

# Get credentials (configures kubectl)
gcloud container clusters get-credentials my-cluster \
  --region us-central1

# List clusters
gcloud container clusters list

# Resize a node pool
gcloud container clusters resize my-cluster \
  --node-pool default-pool --num-nodes 5

# Delete a cluster
gcloud container clusters delete my-cluster --region us-central1
```

## Cloud Storage

### Object operations

```bash
# List buckets
gcloud storage ls

# List objects in a bucket
gcloud storage ls gs://my-bucket/

# Copy file to bucket
gcloud storage cp myfile.txt gs://my-bucket/

# Copy from bucket to local
gcloud storage cp gs://my-bucket/myfile.txt ./

# Recursive copy
gcloud storage cp -r ./local-dir gs://my-bucket/prefix/

# Sync directories (like rsync)
gcloud storage rsync ./local-dir gs://my-bucket/prefix/ --recursive

# Delete objects
gcloud storage rm gs://my-bucket/myfile.txt
gcloud storage rm gs://my-bucket/** # all objects
```

## IAM

### Roles and policies

```bash
# List IAM policy for a project
gcloud projects get-iam-policy my-project-id

# Add a role binding
gcloud projects add-iam-policy-binding my-project-id \
  --member="user:dev@example.com" \
  --role="roles/editor"

# Remove a role binding
gcloud projects remove-iam-policy-binding my-project-id \
  --member="user:dev@example.com" \
  --role="roles/editor"

# List predefined roles
gcloud iam roles list --filter="name:roles/compute"

# Create a service account
gcloud iam service-accounts create my-sa \
  --display-name="My Service Account"

# Create a key for a service account
gcloud iam service-accounts keys create key.json \
  --iam-account my-sa@my-project-id.iam.gserviceaccount.com
```

## Cloud Run

### Deploy and manage

```bash
# Deploy from source (builds with buildpacks)
gcloud run deploy my-service --source . --region us-central1

# Deploy a container image
gcloud run deploy my-service \
  --image gcr.io/my-project/my-image:latest \
  --region us-central1 \
  --allow-unauthenticated

# List services
gcloud run services list

# View logs
gcloud run services logs read my-service --region us-central1

# Set environment variables
gcloud run services update my-service \
  --set-env-vars KEY=value,OTHER=val2
```

## Pub/Sub

### Topics and subscriptions

```bash
# Create a topic
gcloud pubsub topics create my-topic

# Publish a message
gcloud pubsub topics publish my-topic --message "hello world"

# Create a subscription
gcloud pubsub subscriptions create my-sub --topic my-topic

# Pull messages
gcloud pubsub subscriptions pull my-sub --auto-ack --limit 10

# List topics and subscriptions
gcloud pubsub topics list
gcloud pubsub subscriptions list
```

## Logging

### Log exploration

```bash
# Read recent logs
gcloud logging read "resource.type=gce_instance" --limit 20

# Filter by severity
gcloud logging read "severity>=ERROR" --limit 50

# Tail logs in real time
gcloud logging tail "resource.type=cloud_run_revision"

# Read logs for a specific service
gcloud logging read 'resource.type="cloud_run_revision" AND resource.labels.service_name="my-service"' \
  --limit 30
```

## Common Flags

```bash
# Output format: json, yaml, table, csv, value, none
gcloud compute instances list --format="table(name,zone,status)"
gcloud compute instances list --format=json

# Server-side filtering
gcloud compute instances list --filter="status=RUNNING"
gcloud compute instances list --filter="name~'^web-'"

# Override project per-command
gcloud compute instances list --project other-project

# Quiet mode (suppress prompts)
gcloud compute instances delete my-vm --quiet
```

## Tips

- Use `gcloud components update` to keep the CLI up to date.
- Use `gcloud info` to debug configuration and environment issues.
- Use `gcloud interactive` for an autocomplete shell experience.
- Most `create` commands accept `--labels` for organizing resources.
- Add `--verbosity=debug` to any command for detailed request/response logging.
- Use `gcloud config set accessibility/screen_reader true` for accessible output.

## References

- [gcloud CLI Reference](https://cloud.google.com/sdk/gcloud/reference)
- [Google Cloud SDK Documentation](https://cloud.google.com/sdk/docs/)
- [gcloud CLI Cheat Sheet (official)](https://cloud.google.com/sdk/docs/cheatsheet)
- [gcloud CLI Install Guide](https://cloud.google.com/sdk/docs/install)
- [gcloud Filters](https://cloud.google.com/sdk/gcloud/reference/topic/filters)
- [gcloud Formats](https://cloud.google.com/sdk/gcloud/reference/topic/formats)
- [gcloud Configurations](https://cloud.google.com/sdk/gcloud/reference/config)
- [Service Account Authentication](https://cloud.google.com/iam/docs/service-account-overview)
- [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials)
- [gcloud CLI GitHub Repository](https://github.com/google-cloud-sdk-unofficial/google-cloud-sdk)
- [Google Cloud CLI Properties](https://cloud.google.com/sdk/docs/properties)
