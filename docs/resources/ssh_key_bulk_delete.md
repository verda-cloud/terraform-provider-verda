---
page_title: "verda_ssh_key_bulk_delete Resource - Verda Provider"
subcategory: "Compute"
description: |-
  Deletes multiple SSH keys in a single request.
---

# verda_ssh_key_bulk_delete (Resource)

Deletes multiple SSH keys in a single request.

## Example Usage

```terraform
resource "verda_ssh_key_bulk_delete" "cleanup" {
  key_ids = [
    verda_ssh_key.one.id,
    verda_ssh_key.two.id,
  ]
}
```

## Schema

### Required

- `key_ids` (List of String) SSH key IDs to delete.

### Read-Only

- `id` (String) Action identifier.
