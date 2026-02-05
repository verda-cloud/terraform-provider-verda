---
page_title: "verda_job_action Resource - Verda Provider"
subcategory: "Jobs"
description: |-
  Performs an action on a Verda serverless job deployment.
---

# verda_job_action (Resource)

Performs an action on a serverless job deployment (pause, resume, purge queue).

## Example Usage

```terraform
resource "verda_job_action" "pause_job" {
  job_name = verda_serverless_job.batch.name
  action   = "pause"
}
```

## Schema

### Required

- `action` (String) Action to perform: `pause`, `resume`, or `purge_queue`.
- `job_name` (String) Job deployment name.

### Read-Only

- `id` (String) Action identifier.
