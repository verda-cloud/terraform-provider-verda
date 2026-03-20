---
page_title: "verda_serverless_job Data Source - Verda Provider"
subcategory: "Containers"
description: |-
  Reads a Verda serverless job deployment by name.
---

# verda_serverless_job (Data Source)

Reads a Verda serverless job deployment by name using the serverless jobs GET endpoint.

## Example Usage

```terraform
data "verda_serverless_job" "example" {
  name = "existing-job"
}
```

## Schema

### Required

- `name` (String) Name of the serverless job deployment.

### Read-Only

- `compute` (Attributes) Compute resources for the job deployment. See [below for nested schema](#nestedatt--compute).
- `container_registry_settings` (Attributes) Container registry authentication settings. See [below for nested schema](#nestedatt--container_registry_settings).
- `containers` (Attributes List) List of containers in the job deployment. See [below for nested schema](#nestedatt--containers).
- `created_at` (String) Timestamp when the job deployment was created.
- `endpoint_base_url` (String) Base URL for the job deployment endpoint.
- `scaling` (Attributes) Scaling configuration for the job deployment. See [below for nested schema](#nestedatt--scaling).

<a id="nestedatt--compute"></a>
### Nested Schema for `compute`

Read-Only:

- `name` (String) GPU type.
- `size` (Number) Number of GPUs.

<a id="nestedatt--container_registry_settings"></a>
### Nested Schema for `container_registry_settings`

Read-Only:

- `credentials` (String) Name of the registry credentials resource.
- `is_private` (String) Whether the registry is private (`true` or `false`).

<a id="nestedatt--containers"></a>
### Nested Schema for `containers`

Read-Only:

- `entrypoint_overrides` (Attributes) Override container entrypoint and command. See [below](#nestedatt--containers--entrypoint_overrides).
- `env` (Attributes List) Environment variables. See [below](#nestedatt--containers--env).
- `exposed_port` (Number) Port exposed by the container.
- `healthcheck` (Attributes) Healthcheck configuration. See [below](#nestedatt--containers--healthcheck).
- `image` (String) Container image.
- `volume_mounts` (Attributes List) Volume mounts. See [below](#nestedatt--containers--volume_mounts).

<a id="nestedatt--containers--entrypoint_overrides"></a>
### Nested Schema for `containers.entrypoint_overrides`

Read-Only:

- `cmd` (List of String) Custom command array.
- `enabled` (Boolean) Whether to override the entrypoint.
- `entrypoint` (List of String) Custom entrypoint array.

<a id="nestedatt--containers--env"></a>
### Nested Schema for `containers.env`

Read-Only:

- `name` (String) Name of the environment variable.
- `type` (String) Type of environment variable (`plain` or `secret`).
- `value_or_reference_to_secret` (String) Value for plain env vars or secret name for secret env vars.

<a id="nestedatt--containers--healthcheck"></a>
### Nested Schema for `containers.healthcheck`

Read-Only:

- `enabled` (String) Whether healthcheck is enabled (`true` or `false`).
- `path` (String) Path for healthcheck.
- `port` (String) Port for healthcheck.

<a id="nestedatt--containers--volume_mounts"></a>
### Nested Schema for `containers.volume_mounts`

Read-Only:

- `mount_path` (String) Path where volume will be mounted in container.
- `secret_name` (String) Name of secret when present.
- `size_in_mb` (Number) Size in MB when present.
- `type` (String) Type of volume.
- `volume_id` (String) Volume ID when present.

<a id="nestedatt--scaling"></a>
### Nested Schema for `scaling`

Read-Only:

- `deadline_seconds` (Number) Request deadline in seconds.
- `max_replica_count` (Number) Maximum number of replicas.
- `queue_message_ttl_seconds` (Number) Queue message TTL in seconds.
