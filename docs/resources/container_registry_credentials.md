---
page_title: "verda_container_registry_credentials Resource - Verda Provider"
subcategory: "Containers"
description: |-
  Manages credentials for accessing private container registries.
---

# verda_container_registry_credentials (Resource)

Manages credentials for accessing private container registries in Verda container deployments. Create credentials before referencing them in container deployments.

## Example Usage

### Docker Hub

```terraform
resource "verda_container_registry_credentials" "dockerhub" {
  name         = "dockerhub"
  type         = "dockerhub"
  username     = var.docker_username
  access_token = var.docker_token
}
```

### GitHub Container Registry

```terraform
resource "verda_container_registry_credentials" "ghcr" {
  name         = "ghcr"
  type         = "ghcr"
  username     = var.github_username
  access_token = var.github_token
}
```

### Google Container Registry

```terraform
resource "verda_container_registry_credentials" "gcr" {
  name                = "gcr"
  type                = "gcr"
  service_account_key = file("service-account.json")
}
```

### Amazon ECR

```terraform
resource "verda_container_registry_credentials" "ecr" {
  name              = "ecr"
  type              = "ecr"
  access_key_id     = var.aws_access_key_id
  secret_access_key = var.aws_secret_access_key
  region            = "us-east-1"
  ecr_repo          = "123456789012.dkr.ecr.us-east-1.amazonaws.com"
}
```

### Using with Container Deployment

```terraform
resource "verda_container_registry_credentials" "private" {
  name         = "my-registry"
  type         = "dockerhub"
  username     = var.docker_username
  access_token = var.docker_token
}

resource "verda_container" "app" {
  name = "private-app"

  container_registry_settings = {
    is_private  = "true"
    credentials = verda_container_registry_credentials.private.name
  }

  compute = {
    name = "H100"
    size = 1
  }

  scaling = {
    min_replica_count               = 1
    max_replica_count               = 5
    queue_message_ttl_seconds       = 300
    concurrent_requests_per_replica = 10

    scale_down_policy = { delay_seconds = 60 }
    scale_up_policy   = { delay_seconds = 10 }
    queue_load        = { threshold = 0.5 }
  }

  containers = [
    {
      image        = "myorg/private-image:latest"
      exposed_port = 8080
    }
  ]
}
```

-> **Tip:** Store credentials in Terraform variables or a secrets manager rather than hardcoding them in configuration files.

## Schema

### Required

- `name` (String) Name of the registry credentials.
- `type` (String) Registry type: `dockerhub`, `gcr`, `ghcr`, `ecr`, `scaleway`, or `custom`.

### Optional (varies by registry type)

**Docker Hub / GHCR:**
- `username` (String, Sensitive) Registry username.
- `access_token` (String, Sensitive) Access token or password.

**Google Container Registry:**
- `service_account_key` (String, Sensitive) Service account key JSON.

**Amazon ECR:**
- `access_key_id` (String, Sensitive) AWS Access Key ID.
- `secret_access_key` (String, Sensitive) AWS Secret Access Key.
- `region` (String) AWS region.
- `ecr_repo` (String) ECR repository URL.

**Scaleway:**
- `scaleway_domain` (String) Scaleway registry domain.
- `scaleway_uuid` (String) Scaleway namespace UUID.

**Generic:**
- `docker_config_json` (String, Sensitive) Docker config.json content.

### Read-Only

- `created_at` (String) Creation timestamp in ISO 8601 format.

## Import

Existing credentials can be imported using the credentials name:

```shell
terraform import verda_container_registry_credentials.example <credentials-name>
```
