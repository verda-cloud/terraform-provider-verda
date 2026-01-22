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
  instance_type = "gpu-medium"
  image         = "ubuntu-22.04"
  hostname      = "ml-workstation"
  description   = "Machine learning development instance"
  location      = "FIN-01"

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

## Schema

### Optional

- `client_id` (String) Verda OAuth2 Client ID. Can also be set via the `VERDA_CLIENT_ID` environment variable.
- `client_secret` (String, Sensitive) Verda OAuth2 Client Secret. Can also be set via the `VERDA_CLIENT_SECRET` environment variable.
- `base_url` (String) Verda API Base URL. Defaults to `https://api.verda.com/v1`. Can also be set via the `VERDA_BASE_URL` environment variable.

## Resources

The Verda provider includes the following resources:

### Compute

- [verda_instance](resources/instance.md) - GPU compute instances
- [verda_ssh_key](resources/ssh_key.md) - SSH keys for instance access
- [verda_startup_script](resources/startup_script.md) - Startup scripts for instance initialization

### Storage

- [verda_volume](resources/volume.md) - Persistent NVMe storage volumes

### Containers

- [verda_container](resources/container.md) - Serverless container deployments with auto-scaling
- [verda_serverless_job](resources/serverless_job.md) - Batch job deployments
- [verda_container_registry_credentials](resources/container_registry_credentials.md) - Private registry authentication
