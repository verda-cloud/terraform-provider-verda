---
page_title: "verda_file_secret Resource - Verda Provider"
subcategory: "Secrets"
description: |-
  Manages a Verda fileset secret for mounting files into deployments.
---

# verda_file_secret (Resource)

Manages a Verda fileset secret. File secrets can be mounted into containers as secret volumes.

## Example Usage

```terraform
resource "verda_file_secret" "config" {
  name = "config-files"

  files = [
    {
      file_name      = "config.json"
      base64_content = base64encode(file("${path.module}/config.json"))
    }
  ]
}
```

## Schema

### Required

- `files` (Attributes List) Files to store in the secret. See [below for nested schema](#nestedatt--files).
- `name` (String) Name of the file secret.

### Read-Only

- `created_at` (String) Creation timestamp.
- `file_names` (List of String) File names stored in the secret.
- `secret_type` (String) Secret type as reported by the API.

## Nested Schema

### `files`

Required:

- `base64_content` (String, Sensitive) Base64-encoded file content.
- `file_name` (String) File name to store.
