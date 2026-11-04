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

- [verda_instance](resources/instance.md) - GPU compute instances
- [verda_ssh_key](resources/ssh_key.md) - SSH keys for instance access
- [verda_startup_script](resources/startup_script.md) - Startup scripts for instance initialization

### Storage

- [verda_volume](resources/volume.md) - Persistent NVMe storage volumes

### Networking

Verda Private Networks are rolling out across the Verda fleet, starting with `FIN-03`.

- [verda_private_network](resources/private_network.md) - Private networks
- [verda_subnet](resources/subnet.md) - Subnets of a private network, per location
- [verda_network_route](resources/network_route.md) - Routes a subnet's traffic through an instance, e.g. a NAT instance
- [verda_private_network_default](resources/private_network_default.md) - The project's default private network
- [verda_subnet_default](resources/subnet_default.md) - A private network's default subnet in a location

### Containers

- [verda_container](resources/container.md) - Serverless container deployments with auto-scaling
- [verda_serverless_job](resources/serverless_job.md) - Batch job deployments
- [verda_container_registry_credentials](resources/container_registry_credentials.md) - Private registry authentication

## Data Sources

- [verda_private_network](data-sources/private_network.md) - Look up a private network by ID, name, or as the project default
- [verda_subnet](data-sources/subnet.md) - Look up a private network subnet by ID, name, or as a location's default
