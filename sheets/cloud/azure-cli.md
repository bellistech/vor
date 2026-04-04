# Azure CLI (Microsoft Azure Command-Line Interface)

Cross-platform tool for creating and managing Azure resources from the terminal.

## Authentication

### Login and accounts

```bash
# Interactive browser login
az login

# Login with a service principal
az login --service-principal \
  --username APP_ID \
  --password CLIENT_SECRET \
  --tenant TENANT_ID

# Login with managed identity (from Azure VM/container)
az login --identity

# Show current account
az account show

# List all subscriptions
az account list --output table

# Set active subscription
az account set --subscription "My Subscription"
# or by ID
az account set --subscription 00000000-0000-0000-0000-000000000000

# Logout
az logout
```

## Resource Groups

### Manage resource groups

```bash
# List resource groups
az group list --output table

# Create a resource group
az group create --name my-rg --location eastus

# Delete a resource group and all its resources
az group delete --name my-rg --yes --no-wait

# Show details
az group show --name my-rg
```

## Virtual Machines

### Create and manage VMs

```bash
# Create a VM (Ubuntu, generates SSH keys if missing)
az vm create \
  --resource-group my-rg \
  --name my-vm \
  --image Ubuntu2204 \
  --size Standard_B2s \
  --admin-username azureuser \
  --generate-ssh-keys

# List VMs
az vm list --output table
az vm list --resource-group my-rg --output table

# Show VM details
az vm show --resource-group my-rg --name my-vm --show-details

# Start / stop / restart / deallocate
az vm start --resource-group my-rg --name my-vm
az vm stop --resource-group my-rg --name my-vm
az vm restart --resource-group my-rg --name my-vm
az vm deallocate --resource-group my-rg --name my-vm  # stops billing

# Delete a VM
az vm delete --resource-group my-rg --name my-vm --yes

# SSH into a VM
az ssh vm --resource-group my-rg --name my-vm

# List available VM sizes in a region
az vm list-sizes --location eastus --output table

# List available images
az vm image list --output table
az vm image list --offer Ubuntu --all --output table
```

## Networking

### Virtual networks and security

```bash
# Create a virtual network
az network vnet create \
  --resource-group my-rg \
  --name my-vnet \
  --address-prefix 10.0.0.0/16 \
  --subnet-name default \
  --subnet-prefix 10.0.0.0/24

# List vnets
az network vnet list --resource-group my-rg --output table

# Create a network security group
az network nsg create --resource-group my-rg --name my-nsg

# Add an NSG rule (allow SSH)
az network nsg rule create \
  --resource-group my-rg \
  --nsg-name my-nsg \
  --name allow-ssh \
  --priority 1000 \
  --protocol Tcp \
  --destination-port-ranges 22 \
  --access Allow

# Create a public IP
az network public-ip create \
  --resource-group my-rg \
  --name my-ip \
  --sku Standard \
  --allocation-method Static

# Show a public IP address
az network public-ip show \
  --resource-group my-rg --name my-ip \
  --query ipAddress --output tsv
```

## Storage

### Accounts, containers, and blobs

```bash
# Create a storage account
az storage account create \
  --name mystorageacct \
  --resource-group my-rg \
  --location eastus \
  --sku Standard_LRS

# List storage accounts
az storage account list --output table

# Get connection string
az storage account show-connection-string \
  --name mystorageacct --resource-group my-rg

# Create a blob container
az storage container create \
  --name my-container \
  --account-name mystorageacct

# Upload a blob
az storage blob upload \
  --account-name mystorageacct \
  --container-name my-container \
  --file myfile.txt \
  --name myfile.txt

# List blobs
az storage blob list \
  --account-name mystorageacct \
  --container-name my-container \
  --output table

# Download a blob
az storage blob download \
  --account-name mystorageacct \
  --container-name my-container \
  --name myfile.txt \
  --file downloaded.txt
```

## AKS (Azure Kubernetes Service)

### Cluster management

```bash
# Create a cluster
az aks create \
  --resource-group my-rg \
  --name my-cluster \
  --node-count 3 \
  --node-vm-size Standard_B2s \
  --generate-ssh-keys

# Get credentials (configures kubectl)
az aks get-credentials --resource-group my-rg --name my-cluster

# List clusters
az aks list --output table

# Scale a node pool
az aks scale --resource-group my-rg --name my-cluster --node-count 5

# Delete a cluster
az aks delete --resource-group my-rg --name my-cluster --yes
```

## ACR (Azure Container Registry)

### Container registry operations

```bash
# Create a registry
az acr create --resource-group my-rg --name myregistry --sku Basic

# Login to registry (for docker push/pull)
az acr login --name myregistry

# Build an image in ACR
az acr build --registry myregistry --image myapp:latest .

# List repositories
az acr repository list --name myregistry --output table

# List tags for a repository
az acr repository show-tags --name myregistry --repository myapp
```

## Key Vault

### Secrets management

```bash
# Create a key vault
az keyvault create --name my-vault --resource-group my-rg --location eastus

# Set a secret
az keyvault secret set --vault-name my-vault --name my-secret --value "s3cr3t"

# Get a secret value
az keyvault secret show --vault-name my-vault --name my-secret \
  --query value --output tsv

# List secrets
az keyvault secret list --vault-name my-vault --output table

# Delete a secret
az keyvault secret delete --vault-name my-vault --name my-secret
```

## Web Apps (App Service)

### Deploy and manage

```bash
# Create an app service plan
az appservice plan create \
  --name my-plan --resource-group my-rg --sku B1 --is-linux

# Create a web app
az webapp create \
  --resource-group my-rg \
  --plan my-plan \
  --name my-webapp \
  --runtime "NODE:18-lts"

# Deploy from local zip
az webapp deploy --resource-group my-rg --name my-webapp \
  --src-path app.zip --type zip

# Set app settings (env vars)
az webapp config appsettings set \
  --resource-group my-rg --name my-webapp \
  --settings KEY=value OTHER=val2

# Tail logs
az webapp log tail --resource-group my-rg --name my-webapp
```

## Common Patterns

### Queries and output

```bash
# Output formats: json (default), table, tsv, yaml, jsonc, none
az vm list --output table

# JMESPath queries
az vm list --query "[].{Name:name,RG:resourceGroup,Status:powerState}" --output table

# Filter with JMESPath
az vm list --query "[?location=='eastus']" --output table

# Get a single value
az vm show -g my-rg -n my-vm --query "hardwareProfile.vmSize" --output tsv

# Find available commands
az find "vm create"
```

## Tips

- Use `az interactive` for an autocomplete shell with inline docs.
- Use `--no-wait` on long-running operations to return immediately.
- Use `az resource list --resource-group my-rg` to see all resources in a group.
- Use `az account list-locations --output table` to find region names.
- Most commands accept `--tags` for resource tagging.
- Use `az extension add --name <ext>` to add CLI extensions (e.g., `aks-preview`).

## See Also

- aws-cli
- gcloud
- terraform
- kubernetes
- docker

## References

- [Azure CLI Reference](https://learn.microsoft.com/en-us/cli/azure/reference-index)
- [Azure CLI Overview](https://learn.microsoft.com/en-us/cli/azure/)
- [Azure CLI Install Guide](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli)
- [JMESPath Queries in Azure CLI](https://learn.microsoft.com/en-us/cli/azure/query-azure-cli)
- [Azure CLI Output Formats](https://learn.microsoft.com/en-us/cli/azure/format-output-azure-cli)
- [Azure CLI Configuration](https://learn.microsoft.com/en-us/cli/azure/azure-cli-configuration)
- [Azure CLI Service Principal Authentication](https://learn.microsoft.com/en-us/cli/azure/authenticate-azure-cli-service-principal)
- [Azure CLI Extensions](https://learn.microsoft.com/en-us/cli/azure/azure-cli-extensions-overview)
- [Azure CLI Interactive Mode](https://learn.microsoft.com/en-us/cli/azure/interactive-azure-cli)
- [Azure CLI GitHub Repository](https://github.com/Azure/azure-cli)
- [JMESPath Specification](https://jmespath.org/specification.html)
