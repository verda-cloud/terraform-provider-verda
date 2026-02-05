---
page_title: "verda_startup_script_bulk_delete Resource - Verda Provider"
subcategory: "Compute"
description: |-
  Deletes multiple startup scripts in a single request.
---

# verda_startup_script_bulk_delete (Resource)

Deletes multiple startup scripts in a single request.

## Example Usage

```terraform
resource "verda_startup_script_bulk_delete" "cleanup" {
  script_ids = [
    verda_startup_script.one.id,
    verda_startup_script.two.id,
  ]
}
```

## Schema

### Required

- `script_ids` (List of String) Startup script IDs to delete.

### Read-Only

- `id` (String) Action identifier.
