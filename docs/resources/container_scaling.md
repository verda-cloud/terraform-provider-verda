---
page_title: "verda_container_scaling Resource - Verda Provider"
subcategory: "Containers"
description: |-
  Manages scaling settings for a Verda container deployment.
---

# verda_container_scaling (Resource)

Manages scaling settings for a Verda container deployment using the scaling API endpoint.

## Example Usage

```terraform
resource "verda_container_scaling" "api_scaling" {
  deployment_name = verda_container.api.name

  min_replica_count               = 1
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

  cpu_utilization = {
    enabled   = true
    threshold = 80
  }
}
```

## Schema

### Required

- `deployment_name` (String) Deployment name.
- `min_replica_count` (Number) Minimum replicas.
- `max_replica_count` (Number) Maximum replicas.
- `queue_message_ttl_seconds` (Number) Queue message TTL in seconds.
- `concurrent_requests_per_replica` (Number) Max concurrent requests per replica.
- `scale_down_policy` (Attributes) Scale down policy. See [below for nested schema](#nestedatt--scale_down_policy).
- `scale_up_policy` (Attributes) Scale up policy. See [below for nested schema](#nestedatt--scale_up_policy).
- `queue_load` (Attributes) Queue load trigger. See [below for nested schema](#nestedatt--queue_load).

### Optional

- `cpu_utilization` (Attributes) CPU utilization trigger. See [below for nested schema](#nestedatt--cpu_utilization).
- `gpu_utilization` (Attributes) GPU utilization trigger. See [below for nested schema](#nestedatt--gpu_utilization).

## Nested Schema

### `scale_down_policy`

Required:

- `delay_seconds` (Number)

### `scale_up_policy`

Required:

- `delay_seconds` (Number)

### `queue_load`

Required:

- `threshold` (Number)

### `cpu_utilization`

Required:

- `enabled` (Boolean)
- `threshold` (Number)

### `gpu_utilization`

Required:

- `enabled` (Boolean)
- `threshold` (Number)
