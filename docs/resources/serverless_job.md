---
page_title: "verda_serverless_job Resource - Verda Provider"
subcategory: "Containers"
description: |-
  Manages serverless GPU job deployments for batch processing workloads.
---

# verda_serverless_job (Resource)

Manages serverless GPU job deployments for batch processing workloads. Unlike container deployments, serverless jobs are optimized for tasks that run to completion, such as training jobs, data processing, or batch inference.

## Example Usage

### Basic Serverless Job

```terraform
resource "verda_serverless_job" "processor" {
  name = "batch-processor"

  compute = {
    name = "H100"
    size = 1
  }

  scaling = {
    max_replica_count         = 10
    queue_message_ttl_seconds = 3600
    deadline_seconds          = 1800
  }

  containers = [
    {
      image        = "myorg/batch-job:latest"
      exposed_port = 8000
    }
  ]
}
```

### ML Training Job with Private Registry

```terraform
resource "verda_container_registry_credentials" "docker" {
  name         = "dockerhub"
  type         = "dockerhub"
  username     = var.docker_username
  access_token = var.docker_token
}

resource "verda_serverless_job" "training" {
  name = "model-training"

  compute = {
    name = "A100"
    size = 4
  }

  scaling = {
    max_replica_count         = 20
    queue_message_ttl_seconds = 7200
    deadline_seconds          = 3600
  }

  container_registry_settings = {
    is_private  = "true"
    credentials = verda_container_registry_credentials.docker.name
  }

  containers = [
    {
      image        = "myorg/ml-training:v2"
      exposed_port = 8080

      healthcheck = {
        enabled = "true"
        port    = "8080"
        path    = "/health"
      }

      entrypoint_overrides = {
        enabled    = true
        entrypoint = ["python"]
        cmd        = ["train.py", "--epochs", "100"]
      }

      env = [
        {
          type                         = "plain"
          name                         = "BATCH_SIZE"
          value_or_reference_to_secret = "32"
        },
        {
          type                         = "secret"
          name                         = "WANDB_API_KEY"
          value_or_reference_to_secret = "wandb-key"
        }
      ]

      volume_mounts = [
        {
          type       = "scratch"
          mount_path = "/tmp/data"
          size_in_mb = 51200
        }
      ]
    }
  ]
}
```

### Data Processing Job

```terraform
resource "verda_serverless_job" "etl" {
  name = "data-etl"

  compute = {
    name = "H100"
    size = 2
  }

  scaling = {
    max_replica_count         = 50
    queue_message_ttl_seconds = 1800
    deadline_seconds          = 900
  }

  containers = [
    {
      image        = "myorg/etl-pipeline:latest"
      exposed_port = 8000

      env = [
        {
          type                         = "plain"
          name                         = "WORKERS"
          value_or_reference_to_secret = "8"
        },
        {
          type                         = "secret"
          name                         = "DB_CONNECTION"
          value_or_reference_to_secret = "db-conn-string"
        }
      ]
    }
  ]
}
```

~> **Note:** Serverless jobs differ from container deployments in that they don't maintain minimum replicas and are designed for workloads that complete and exit.

## Schema

### Required

- `compute` (Attributes) Compute resources for the job. See [below for nested schema](#nestedatt--compute).
- `containers` (Attributes List) List of containers in the job. See [below for nested schema](#nestedatt--containers).
- `name` (String) Name of the serverless job deployment.
- `scaling` (Attributes) Scaling configuration. See [below for nested schema](#nestedatt--scaling).

### Optional

- `container_registry_settings` (Attributes) Private registry authentication. See [below for nested schema](#nestedatt--container_registry_settings).

### Read-Only

- `created_at` (String) Creation timestamp in ISO 8601 format.
- `endpoint_base_url` (String) Base URL for the job deployment endpoint.

<a id="nestedatt--compute"></a>
### Nested Schema for `compute`

Required:

- `name` (String) GPU type (e.g., `H100`, `A100`).
- `size` (Number) Number of GPUs per job instance.

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
- `entrypoint` (List of String) Custom entrypoint (e.g., `["python"]`).

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

- `path` (String) HTTP path for healthcheck.
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

- `max_replica_count` (Number) Maximum concurrent job instances.
- `queue_message_ttl_seconds` (Number) How long jobs stay in queue before expiring.

Optional:

- `deadline_seconds` (Number) Maximum time for a job to complete.

<a id="nestedatt--container_registry_settings"></a>
### Nested Schema for `container_registry_settings`

Optional:

- `credentials` (String) Name of the registry credentials resource.
- `is_private` (String) Whether the registry is private (`true` or `false`).

## Import

Existing job deployments can be imported using the deployment name:

```shell
terraform import verda_serverless_job.example <job-deployment-name>
```
