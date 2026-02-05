---
page_title: "verda_container_environment_variables Resource - Verda Provider"
subcategory: "Containers"
description: |-
  Manages environment variables for a Verda container deployment.
---

# verda_container_environment_variables (Resource)

Manages environment variables for a container inside a deployment.

## Example Usage

```terraform
resource "verda_container_environment_variables" "api_env" {
  deployment_name = verda_container.api.name
  container_name  = "api"

  env = [
    {
      type                         = "plain"
      name                         = "LOG_LEVEL"
      value_or_reference_to_secret = "info"
    },
    {
      type                         = "secret"
      name                         = "API_KEY"
      value_or_reference_to_secret = "api-key-secret"
    }
  ]
}
```

## Schema

### Required

- `deployment_name` (String) Deployment name.
- `container_name` (String) Container name within the deployment.
- `env` (Attributes List) Environment variables. See [below for nested schema](#nestedatt--env).

### Read-Only

- `id` (String) Composite identifier.

## Nested Schema

### `env`

Required:

- `name` (String) Environment variable name.
- `type` (String) Environment variable type (`plain` or `secret`).
- `value_or_reference_to_secret` (String, Sensitive) Value or secret reference.
