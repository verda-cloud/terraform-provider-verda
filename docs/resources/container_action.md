---
page_title: "verda_container_action Resource - Verda Provider"
subcategory: "Containers"
description: |-
  Performs an action on a Verda container deployment.
---

# verda_container_action (Resource)

Performs an action on a container deployment (pause, resume, restart, purge queue).

## Example Usage

```terraform
resource "verda_container_action" "restart_api" {
  deployment_name = verda_container.api.name
  action          = "restart"
}
```

## Schema

### Required

- `action` (String) Action to perform: `pause`, `resume`, `restart`, or `purge_queue`.
- `deployment_name` (String) Deployment name.

### Read-Only

- `id` (String) Action identifier.
