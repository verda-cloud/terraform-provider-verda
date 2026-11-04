---
page_title: "verda_subnet_default Resource - Verda Provider"
subcategory: "Networking"
description: |-
  Sets which subnet is a private network's default in one location.
---

# verda_subnet_default (Resource)

Sets which subnet is a private network's default in one location. This is a singleton: declare at most one per private network and location.

-> **Note:** Verda Private Networks are rolling out across the Verda fleet: they are available in `FIN-03` today, with more locations coming. The locations API reports `is_private_networks_enabled` for each location.

## Example Usage

```terraform
resource "verda_private_network" "main" {
  name = "main"
}

resource "verda_subnet" "fin03" {
  private_network_id = verda_private_network.main.id
  location           = "FIN-03"
  name               = "fin03"
  cidr               = "10.20.0.0/24"
}

resource "verda_subnet_default" "fin03" {
  private_network_id = verda_private_network.main.id
  location           = "FIN-03"
  subnet_id          = verda_subnet.fin03.id
}
```

## Behavior

- The subnet must belong to `private_network_id` and be in `location`; this is checked on apply.
- Changing `subnet_id` makes another subnet the default, in place.
- If the default is changed outside Terraform, the next plan shows the drift on `subnet_id`. If the private network has no default subnet in the location, the resource drops out of state on refresh and the next apply sets the default again.
- Destroying this resource only removes it from Terraform state and shows a warning. The private network keeps its current default subnet in that location.

## Schema

### Required

- `location` (String) Location code, e.g. `FIN-03`. Changing it replaces the resource.
- `subnet_id` (String) ID of the subnet to make the default. It must belong to `private_network_id` and be in `location`. Can be changed in place.
- `private_network_id` (String) ID of the private network. Changing it replaces the resource.

### Read-Only

- `id` (String) `<private_network_id>/<location>`.

## Import

The current default subnet can be imported using the private network ID and location, separated by `/`:

```shell
terraform import verda_subnet_default.fin03 <private_network_id>/<location>
```

For example:

```shell
terraform import verda_subnet_default.fin03 pn_abc123/FIN-03
```
