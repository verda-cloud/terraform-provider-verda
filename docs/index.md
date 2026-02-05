---
page_title: "Provider: Verda Cloud"
description: |-
  The Verda provider enables Terraform to manage Verda Cloud GPU compute instances, serverless containers, and storage volumes.
---

# Verda Provider

The Verda provider enables you to manage [Verda Cloud](https://verda.com) infrastructure using Terraform. Verda Cloud provides high-performance GPU compute instances, serverless container deployments, and NVMe storage optimized for AI/ML workloads.

Use this provider to:

- Provision GPU-accelerated compute instances
- Deploy serverless containers with auto-scaling
- Manage persistent NVMe storage volumes
- Automate infrastructure with startup scripts

## Example Usage

```terraform
terraform {
  required_providers {
    verda = {
      source  = "verda-cloud/verda"
      version = "~> 1.0"
    }
  }
}

provider "verda" {
  client_id     = var.verda_client_id
  client_secret = var.verda_client_secret
}

# Upload your SSH key
resource "verda_ssh_key" "main" {
  name       = "my-ssh-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

# Create a GPU compute instance
resource "verda_instance" "gpu_workstation" {
  instance_type = "1B200.30V"
  image         = "ubuntu-24.04-cuda-12.8-open-docker"
  hostname      = "ml-workstation"
  description   = "Machine learning development instance"
  location      = "FIN-03"

  ssh_key_ids = [verda_ssh_key.main.id]
}
```

## Authentication

The Verda provider requires OAuth2 credentials (Client ID and Client Secret) to authenticate with the Verda Cloud API.

### Option 1: Provider Configuration

```terraform
provider "verda" {
  client_id     = "your-client-id"
  client_secret = "your-client-secret"
}
```

### Option 2: Environment Variables (Recommended)

Set credentials as environment variables to keep them out of your Terraform configuration:

```bash
export VERDA_CLIENT_ID="your-client-id"
export VERDA_CLIENT_SECRET="your-client-secret"
```

Then use an empty provider block:

```terraform
provider "verda" {}
```

-> **Tip:** Environment variables are the recommended approach for production deployments and CI/CD pipelines.

## API Reference

To discover available instance types, images, and locations, use the Verda API:

- **Instance Types**: `GET https://api.verda.com/v1/instance-types` - Lists all available GPU instance types with specifications
- **Images**: `GET https://api.verda.com/v1/images` - Lists available OS images with CUDA versions
- **Locations**: `GET https://api.verda.com/v1/locations` - Lists available data center locations

For full API documentation, visit: [Verda API Reference](https://api.verda.com/v1/docs)

## Schema

### Optional

- `client_id` (String) Verda OAuth2 Client ID. Can also be set via the `VERDA_CLIENT_ID` environment variable.
- `client_secret` (String, Sensitive) Verda OAuth2 Client Secret. Can also be set via the `VERDA_CLIENT_SECRET` environment variable.
- `base_url` (String) Verda API Base URL. Defaults to `https://api.verda.com/v1`. Can also be set via the `VERDA_BASE_URL` environment variable.

## Resources

The Verda provider includes the following resources:

### Compute

- [verda_cluster](resources/cluster.md) - GPU clusters
- [verda_instance](resources/instance.md) - GPU compute instances
- [verda_instance_action](resources/instance_action.md) - Instance actions
- [verda_ssh_key](resources/ssh_key.md) - SSH keys for instance access
- [verda_ssh_key_bulk_delete](resources/ssh_key_bulk_delete.md) - Bulk SSH key deletion
- [verda_startup_script](resources/startup_script.md) - Startup scripts for instance initialization
- [verda_startup_script_bulk_delete](resources/startup_script_bulk_delete.md) - Bulk startup script deletion

### Storage

- [verda_volume](resources/volume.md) - Persistent NVMe storage volumes
- [verda_volume_action](resources/volume_action.md) - Volume actions

### Containers

- [verda_container](resources/container.md) - Serverless container deployments with auto-scaling
- [verda_container_scaling](resources/container_scaling.md) - Deployment scaling settings
- [verda_container_environment_variables](resources/container_environment_variables.md) - Deployment environment variables
- [verda_container_action](resources/container_action.md) - Deployment actions
- [verda_serverless_job](resources/serverless_job.md) - Batch job deployments
- [verda_job_action](resources/job_action.md) - Job deployment actions
- [verda_container_registry_credentials](resources/container_registry_credentials.md) - Private registry authentication

### Secrets

- [verda_secret](resources/secret.md) - Secrets for deployments
- [verda_file_secret](resources/file_secret.md) - Fileset secrets for mounts

## Data Sources

### Account and Locations

- `verda_balance` - Account balance
- `verda_locations` - Available locations

### Images and Types

- `verda_images` - Instance images
- `verda_cluster_images` - Cluster images
- `verda_instance_types` - Instance types
- `verda_instance_type_price_history` - Instance price history
- `verda_container_types` - Container types
- `verda_cluster_types` - Cluster types
- `verda_volume_types` - Volume types
- `verda_serverless_compute_resources` - Serverless compute resources

### Availability

- `verda_instance_availability` - Instance availability by location
- `verda_instance_type_availability` - Instance type availability
- `verda_cluster_availability` - Cluster availability by location
- `verda_cluster_type_availability` - Cluster type availability

### Long-Term Periods

- `verda_long_term_periods` - All long-term periods
- `verda_long_term_instance_periods` - Instance long-term periods
- `verda_long_term_cluster_periods` - Cluster long-term periods

### Resources

- `verda_instances` - Instances list
- `verda_volumes` - Volumes list
- `verda_volumes_trash` - Volumes in trash
- `verda_clusters` - Clusters list
- `verda_container_deployments` - Container deployments list
- `verda_job_deployments` - Job deployments list
- `verda_container_registry_credentials` - Registry credentials list
- `verda_ssh_keys` - SSH keys list
- `verda_startup_scripts` - Startup scripts list
- `verda_secrets` - Secrets list
- `verda_file_secrets` - File secrets list

### Deployment Insights

- `verda_container_environment_variables` - Container env vars
- `verda_container_deployment_status` - Container deployment status
- `verda_container_deployment_replicas` - Container deployment replicas
- `verda_container_deployment_scaling` - Container deployment scaling
- `verda_job_deployment_status` - Job deployment status
- `verda_job_deployment_scaling` - Job deployment scaling

### Deprecated

- `verda_instance_types_deprecated` - Deprecated instance types endpoint
- `verda_instance_availability_deprecated` - Deprecated availability endpoint
