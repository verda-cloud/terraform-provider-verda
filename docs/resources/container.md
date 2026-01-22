---
page_title: "verda_container Resource - Verda Provider"
subcategory: "Containers"
description: |-
  Manages serverless GPU container deployments with auto-scaling.
---

# verda_container (Resource)

Manages serverless GPU container deployments with automatic scaling. Container deployments are ideal for inference workloads, APIs, and services that need to scale based on demand.

## Example Usage

### Basic Container Deployment

```terraform
resource "verda_container" "api" {
  name = "inference-api"

  compute = {
    name = "H100"
    size = 1
  }

  scaling = {
    min_replica_count               = 0
    max_replica_count               = 5
    queue_message_ttl_seconds       = 300
    concurrent_requests_per_replica = 10

    scale_down_policy = {
      delay_seconds = 60
    }

    scale_up_policy = {
      delay_seconds = 10
    }

    queue_load = {
      threshold = 0.5
    }
  }

  containers = [
    {
      image        = "nginx:latest"
      exposed_port = 80
    }
  ]
}
```

### Private Registry with Healthcheck

```terraform
resource "verda_container_registry_credentials" "docker" {
  name         = "dockerhub"
  type         = "dockerhub"
  username     = var.docker_username
  access_token = var.docker_token
}

resource "verda_container" "ml_api" {
  name = "ml-inference"

  compute = {
    name = "A100"
    size = 2
  }

  scaling = {
    min_replica_count               = 1
    max_replica_count               = 10
    queue_message_ttl_seconds       = 600
    concurrent_requests_per_replica = 5

    scale_down_policy = {
      delay_seconds = 120
    }

    scale_up_policy = {
      delay_seconds = 15
    }

    queue_load = {
      threshold = 0.3
    }
  }

  container_registry_settings = {
    is_private  = "true"
    credentials = verda_container_registry_credentials.docker.name
  }

  containers = [
    {
      image        = "myorg/ml-model:v1"
      exposed_port = 8080

      healthcheck = {
        enabled = "true"
        port    = "8080"
        path    = "/health"
      }

      env = [
        {
          type                         = "plain"
          name                         = "LOG_LEVEL"
          value_or_reference_to_secret = "info"
        },
        {
          type                         = "secret"
          name                         = "API_KEY"
          value_or_reference_to_secret = "api-key-secret"
        }
      ]
    }
  ]
}
```

### Container with Custom Entrypoint and Volumes

```terraform
resource "verda_container" "trainer" {
  name = "model-trainer"

  compute = {
    name = "H100"
    size = 4
  }

  scaling = {
    min_replica_count               = 1
    max_replica_count               = 8
    queue_message_ttl_seconds       = 900
    concurrent_requests_per_replica = 1

    scale_down_policy = {
      delay_seconds = 300
    }

    scale_up_policy = {
      delay_seconds = 5
    }

    queue_load = {
      threshold = 0.2
    }
  }

  containers = [
    {
      image        = "myorg/trainer:latest"
      exposed_port = 8000

      entrypoint_overrides = {
        enabled    = true
        entrypoint = ["/bin/sh", "-c"]
        cmd        = ["python", "serve.py"]
      }

      volume_mounts = [
        {
          type       = "scratch"
          mount_path = "/tmp/cache"
          size_in_mb = 10240
        },
        {
          type       = "shared"
          mount_path = "/models"
          volume_id  = verda_volume.models.id
        }
      ]
    }
  ]
}
```

~> **Note:** When using `min_replica_count = 0`, containers scale to zero when idle, saving costs but adding cold-start latency.

## Schema

### Required

- `compute` (Attributes) Compute resources for the deployment. See [below for nested schema](#nestedatt--compute).
- `containers` (Attributes List) List of containers in the deployment. See [below for nested schema](#nestedatt--containers).
- `name` (String) Name of the container deployment.
- `scaling` (Attributes) Scaling configuration. See [below for nested schema](#nestedatt--scaling).

### Optional

- `container_registry_settings` (Attributes) Private registry authentication. See [below for nested schema](#nestedatt--container_registry_settings).
- `is_spot` (Boolean) Whether to use spot instances. Defaults to `false`.

### Read-Only

- `created_at` (String) Creation timestamp in ISO 8601 format.
- `endpoint_base_url` (String) Base URL for the deployment endpoint.

<a id="nestedatt--compute"></a>
### Nested Schema for `compute`

Required:

- `name` (String) GPU type (e.g., `H100`, `A100`).
- `size` (Number) Number of GPUs per replica.

<a id="nestedatt--containers"></a>
### Nested Schema for `containers`

Required:

- `exposed_port` (Number) Port exposed by the container.
- `image` (String) Container image (e.g., `nginx:latest`).

Optional:

- `entrypoint_overrides` (Attributes) Override container entrypoint. See [below](#nestedatt--containers--entrypoint_overrides).
- `env` (Attributes List) Environment variables. See [below](#nestedatt--containers--env).
- `healthcheck` (Attributes) Healthcheck configuration. See [below](#nestedatt--containers--healthcheck).
- `volume_mounts` (Attributes List) Volume mounts. See [below](#nestedatt--containers--volume_mounts).

<a id="nestedatt--containers--entrypoint_overrides"></a>
### Nested Schema for `containers.entrypoint_overrides`

Required:

- `enabled` (Boolean) Whether to override the entrypoint.

Optional:

- `cmd` (List of String) Custom command arguments.
- `entrypoint` (List of String) Custom entrypoint (e.g., `["/bin/sh", "-c"]`).

<a id="nestedatt--containers--env"></a>
### Nested Schema for `containers.env`

Required:

- `name` (String) Environment variable name.
- `type` (String) Type: `plain` for values, `secret` for secret references.
- `value_or_reference_to_secret` (String) Value or secret name.

<a id="nestedatt--containers--healthcheck"></a>
### Nested Schema for `containers.healthcheck`

Required:

- `enabled` (String) Whether healthcheck is enabled (`true` or `false`).

Optional:

- `path` (String) HTTP path for healthcheck (e.g., `/health`).
- `port` (String) Port for healthcheck.

<a id="nestedatt--containers--volume_mounts"></a>
### Nested Schema for `containers.volume_mounts`

Required:

- `mount_path` (String) Path where volume is mounted in the container.
- `type` (String) Volume type: `scratch`, `memory`, `secret`, or `shared`.

Optional:

- `secret_name` (String) Secret name (required for type `secret`).
- `size_in_mb` (Number) Size in MB (for `scratch` or `memory`).
- `volume_id` (String) Volume ID (required for type `shared`).

<a id="nestedatt--scaling"></a>
### Nested Schema for `scaling`

Required:

- `concurrent_requests_per_replica` (Number) Max concurrent requests per replica.
- `max_replica_count` (Number) Maximum replicas.
- `min_replica_count` (Number) Minimum replicas (0 enables scale-to-zero).
- `queue_load` (Attributes) Queue-based scaling trigger. See [below](#nestedatt--scaling--queue_load).
- `queue_message_ttl_seconds` (Number) Request queue TTL in seconds.
- `scale_down_policy` (Attributes) Scale down configuration. See [below](#nestedatt--scaling--scale_down_policy).
- `scale_up_policy` (Attributes) Scale up configuration. See [below](#nestedatt--scaling--scale_up_policy).

Optional:

- `deadline_seconds` (Number) Request timeout in seconds.

<a id="nestedatt--scaling--queue_load"></a>
### Nested Schema for `scaling.queue_load`

Required:

- `threshold` (Number) Queue load threshold (0.0-1.0) that triggers scaling.

<a id="nestedatt--scaling--scale_down_policy"></a>
### Nested Schema for `scaling.scale_down_policy`

Required:

- `delay_seconds` (Number) Seconds to wait before scaling down.

<a id="nestedatt--scaling--scale_up_policy"></a>
### Nested Schema for `scaling.scale_up_policy`

Required:

- `delay_seconds` (Number) Seconds to wait before scaling up.

<a id="nestedatt--container_registry_settings"></a>
### Nested Schema for `container_registry_settings`

Optional:

- `credentials` (String) Name of the registry credentials resource.
- `is_private` (String) Whether the registry is private (`true` or `false`).

## Import

Existing deployments can be imported using the deployment name:

```shell
terraform import verda_container.example <deployment-name>
```
