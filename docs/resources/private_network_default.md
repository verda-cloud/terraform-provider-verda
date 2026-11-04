---
page_title: "verda_private_network_default Resource - Verda Provider"
subcategory: "Networking"
description: |-
  Sets which private network is the project's default.
---

# verda_private_network_default (Resource)

Sets which private network is the project's default. This is a singleton: declare at most one per project.

-> **Note:** Verda Private Networks are rolling out across the Verda fleet: they are available in `FIN-03` today, with more locations coming. The locations API reports `is_private_networks_enabled` for each location.

## Example Usage

```terraform
resource "verda_private_network" "main" {
  name = "main"
}

resource "verda_private_network_default" "this" {
  private_network_id = verda_private_network.main.id
}
```

## Behavior

- Changing `private_network_id` makes another private network the default, in place.
- If the default is changed outside Terraform, the next plan shows the drift on `private_network_id`. If the project has no default private network at all, the resource drops out of state on refresh and the next apply sets the default again.
- Destroying this resource only removes it from Terraform state and shows a warning. The project keeps its current default private network.

## Schema

### Required

- `private_network_id` (String) ID of the private network to make the project default. Can be changed in place.

### Read-Only

- `id` (String) Always `default`.

## Import

The project's current default can be imported using the fixed ID `default`:

```shell
terraform import verda_private_network_default.this default
```
