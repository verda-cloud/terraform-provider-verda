---
page_title: "verda_serverless_job_scaling Data Source - Verda Provider"
subcategory: "Containers"
description: |-
  Reads the scaling configuration for a Verda serverless job deployment.
---

# verda_serverless_job_scaling (Data Source)

Reads the scaling configuration for a Verda serverless job deployment using the scaling GET endpoint.

## Example Usage

```terraform
data "verda_serverless_job_scaling" "example" {
  name = "existing-job"
}
```

## Schema

### Required

- `name` (String) Name of the serverless job deployment.

### Read-Only

- `deadline_seconds` (Number) Request deadline in seconds.
- `max_replica_count` (Number) Maximum number of replicas.
- `queue_message_ttl_seconds` (Number) Queue message TTL in seconds.
