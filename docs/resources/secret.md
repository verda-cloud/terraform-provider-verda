---
page_title: "verda_secret Resource - Verda Provider"
subcategory: "Secrets"
description: |-
  Manages a Verda secret for deployments.
---

# verda_secret (Resource)

Manages a Verda secret that can be referenced by container and job deployments.

## Example Usage

```terraform
resource "verda_secret" "api_token" {
  name  = "api-token"
  value = var.api_token
}
```

## Schema

### Required

- `name` (String) Name of the secret.
- `value` (String, Sensitive) Secret value.

### Read-Only

- `created_at` (String) Creation timestamp.
- `secret_type` (String) Secret type as reported by the API.
