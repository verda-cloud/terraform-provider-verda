# Terraform Provider Verda - Examples

This directory contains example configurations for all resources provided by the Verda Terraform provider.

## Provider Configuration

See [provider.tf](./provider.tf) for provider configuration.

Using environment variables (recommended):

```bash
export VERDA_CLIENT_ID="your-client-id"
export VERDA_CLIENT_SECRET="your-client-secret"
```

```hcl
provider "verda" {}
```

## Resources

### Instance (verda_instance)

GPU/CPU compute instances for ML workloads.

Example: [resources/verda_instance.tf](./resources/verda_instance.tf)

```hcl
resource "verda_instance" "example" {
  instance_type = "1x3090.1"
  image         = "Ubuntu 22.04 LTS - Nvidia CUDA 12.2"
  hostname      = "my-instance"
  description   = "Example GPU instance"
  ssh_key_ids   = [verda_ssh_key.example.id]
}
```

### Volume (verda_volume)

Persistent storage volumes.

Example: [resources/verda_volume.tf](./resources/verda_volume.tf)

```hcl
resource "verda_volume" "data" {
  name     = "my-data-volume"
  size     = 100
  type     = "NVMe"
  location = "FIN-01"
}
```

### SSH Key (verda_ssh_key)

SSH keys for instance access.

Example: [resources/verda_ssh_key.tf](./resources/verda_ssh_key.tf)

```hcl
resource "verda_ssh_key" "example" {
  name       = "my-ssh-key"
  public_key = file("~/.ssh/id_rsa.pub")
}
```

### Startup Script (verda_startup_script)

Scripts that run when instances start.

Example: [resources/verda_startup_script.tf](./resources/verda_startup_script.tf)

```hcl
resource "verda_startup_script" "example" {
  name = "install-docker"
  script = <<-EOF
    #!/bin/bash
    apt-get update
    curl -fsSL https://get.docker.com | sh
  EOF
}
```

### Container (verda_container)

Containerized application deployments.

Example: [resources/verda_container.tf](./resources/verda_container.tf)

```hcl
resource "verda_container" "nginx" {
  name     = "nginx-server"
  image    = "nginx:latest"
  replicas = 2

  environment = {
    "ENV" = "production"
  }
}
```

### Job (verda_job)

Serverless batch jobs.

Example: [resources/verda_job.tf](./resources/verda_job.tf)

```hcl
resource "verda_job" "processor" {
  name = "data-processor"

  containers {
    name  = "worker"
    image = "myapp/processor:latest"
  }

  scaling {
    min_replicas = 0
    max_replicas = 10
  }
}
```

## Getting Started

1. Install Terraform from [terraform.io](https://www.terraform.io/downloads)

2. Configure credentials:
   ```bash
   export VERDA_CLIENT_ID="your-client-id"
   export VERDA_CLIENT_SECRET="your-client-secret"
   ```

3. Initialize and apply:
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

## Support

- GitHub Issues: [github.com/verda-cloud/terraform-provider-verda/issues](https://github.com/verda-cloud/terraform-provider-verda/issues)
- Documentation: [docs.verda.com](https://docs.verda.com)
