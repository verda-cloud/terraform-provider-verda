---
page_title: "verda_volume_action Resource - Verda Provider"
subcategory: "Storage"
description: |-
  Performs an action on one or more Verda volumes.
---

# verda_volume_action (Resource)

Performs actions on volumes (attach, detach, resize, rename, delete, etc.).

## Example Usage

```terraform
resource "verda_volume_action" "resize" {
  action     = "resize"
  volume_ids = [verda_volume.example.id]
  size       = 200
}
```

## Schema

### Required

- `action` (String) Action to perform.
- `volume_ids` (List of String) Volume IDs to perform the action on.

### Optional

- `instance_id` (String) Instance ID for attach.
- `instance_ids` (List of String) Instance IDs for attach.
- `is_permanent` (Boolean) Delete permanently when action is `delete`.
- `location_code` (String) Target location for clone.
- `name` (String) New volume name.
- `size` (Number) New size in GB.
- `type` (String) Target volume type.

### Read-Only

- `id` (String) Action identifier.
