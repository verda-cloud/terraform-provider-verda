---
page_title: "verda_private_network Resource - Verda Provider"
subcategory: "Networking"
description: |-
  Manages a Verda private network.
---

# verda_private_network (Resource)

Manages a Verda private network. A private network is global: add subnets to it per location with [`verda_subnet`](subnet.md), then place instances on a subnet with the `network` block of [`verda_instance`](instance.md).

-> **Note:** Verda Private Networks are rolling out across the Verda fleet: they are available in `FIN-03` today, with more locations coming. The locations API reports `is_private_networks_enabled` for each location.

Private networks created by Terraform are always custom mode: you create each subnet explicitly. To use the network the platform created for your project, look it up with the [`verda_private_network` data source](../data-sources/private_network.md) instead.

## Example Usage

```terraform
resource "verda_private_network" "example" {
  name = "my-network"
}

resource "verda_subnet" "app" {
  private_network_id = verda_private_network.example.id
  location           = "FIN-03"
  name               = "app"
  cidr               = "10.10.0.0/24"
}
```

### Longer Delete Timeout

Deleting a private network is refused while instances are still on it. Instance deletion finishes in the background, so when you destroy a private network together with its instances, the delete retries until they are gone. `timeouts` bounds how long it keeps trying:

```terraform
resource "verda_private_network" "example" {
  name = "my-network"

  timeouts = {
    delete = "15m"
  }
}
```

~> **Note:** Deleting a private network also deletes its subnets and routes.

## Schema

### Required

- `name` (String) Name of the private network, 1–64 characters, unique within the project. Can be changed in place.

### Optional

- `timeouts` (Attributes) `delete` (default `10m`): how long the delete keeps retrying while instances on the private network finish deleting. On timeout the apply fails with the last API error, which names the instances still attached.

### Read-Only

- `created_at` (String) Creation timestamp in ISO 8601 format.
- `id` (String) Unique private network identifier (`pn_…`).
- `is_default` (Boolean) Whether this is the project's default private network. Read-only here; manage it with [`verda_private_network_default`](private_network_default.md).
- `mode` (String) Subnet allocation mode. Always `custom` for private networks created by Terraform; `auto` for an imported network the platform created.

## Import

Existing private networks can be imported using the private network ID:

```shell
terraform import verda_private_network.example pn_abc123
```
