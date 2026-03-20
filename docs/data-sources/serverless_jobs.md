---
page_title: "verda_serverless_jobs Data Source - Verda Provider"
subcategory: "Containers"
description: |-
  Lists Verda serverless job deployments.
---

# verda_serverless_jobs (Data Source)

Lists Verda serverless job deployments using the serverless jobs list GET endpoint.

## Example Usage

```terraform
data "verda_serverless_jobs" "all" {}
```

## Schema

### Read-Only

- `jobs` (Attributes List) Serverless job deployments returned by the API. See [below for nested schema](#nestedatt--jobs).

<a id="nestedatt--jobs"></a>
### Nested Schema for `jobs`

Read-Only:

- `compute` (Attributes) Compute resources for the job deployment. See [below](#nestedatt--jobs--compute).
- `created_at` (String) Timestamp when the job deployment was created.
- `name` (String) Name of the serverless job deployment.

<a id="nestedatt--jobs--compute"></a>
### Nested Schema for `jobs.compute`

Read-Only:

- `name` (String) GPU type.
- `size` (Number) Number of GPUs.
