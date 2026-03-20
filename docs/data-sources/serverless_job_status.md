---
page_title: "verda_serverless_job_status Data Source - Verda Provider"
subcategory: "Containers"
description: |-
  Reads the runtime status for a Verda serverless job deployment.
---

# verda_serverless_job_status (Data Source)

Reads the runtime status for a Verda serverless job deployment using the status GET endpoint.

## Example Usage

```terraform
data "verda_serverless_job_status" "example" {
  name = "existing-job"
}
```

## Schema

### Required

- `name` (String) Name of the serverless job deployment.

### Read-Only

- `status` (String) Runtime status of the serverless job deployment.
