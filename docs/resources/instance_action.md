---
page_title: "verda_instance_action Resource - Verda Provider"
subcategory: "Compute"
description: |-
  Performs an action on one or more Verda instances.
---

# verda_instance_action (Resource)

Performs an action on one or more instances (start, shutdown, delete, etc.).

## Example Usage

```terraform
resource "verda_instance_action" "shutdown_instances" {
  action       = "shutdown"
  instance_ids = [verda_instance.example.id]
}
```

## Schema

### Required

- `action` (String) Action to perform.
- `instance_ids` (List of String) Instance IDs to perform the action on.

### Optional

- `use_deprecated_endpoint` (Boolean) Use deprecated `/v1/instances/action` endpoint.
- `volume_ids` (List of String) Volume IDs to delete when action is `delete` or `delete_stuck`.

### Read-Only

- `id` (String) Action identifier.
