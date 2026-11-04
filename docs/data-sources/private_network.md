---
page_title: "verda_private_network Data Source - Verda Provider"
subcategory: "Networking"
description: |-
  Looks up a Verda private network by ID, by name, or as the project default.
---

# verda_private_network (Data Source)

Looks up a Verda private network by ID, by name, or as the project default. Set exactly one of `id`, `name` or `default = true`.

-> **Note:** Verda Private Networks are rolling out across the Verda fleet: they are available in `FIN-03` today, with more locations coming. The locations API reports `is_private_networks_enabled` for each location.

## Example Usage

### Platform-Created Default Private Network

Leaving out the `network` block does not put an instance on your default private network: it gets public networking only. To place an instance on the default private network, look it up and use its subnet. This also works for the network the platform creates for your project, which the provider cannot create:

```terraform
data "verda_private_network" "default" {
  default = true
}

data "verda_subnet" "default" {
  private_network_id = data.verda_private_network.default.id
  location           = "FIN-03"
  default            = true
}

resource "verda_instance" "example" {
  instance_type = "1B200.30V"
  image         = "ubuntu-24.04-cuda-12.8-open-docker"
  hostname      = "my-instance"
  description   = "Instance on the default private network"
  location      = "FIN-03"

  ssh_key_ids = [verda_ssh_key.example.id]

  network {
    subnet_id = data.verda_subnet.default.id
  }
}
```

### By Name

```terraform
data "verda_private_network" "prod" {
  name = "prod"
}
```

The lookup fails unless exactly one private network matches.

## Schema

### Optional

- `default` (Boolean) Set to `true` to look up the project's default private network.
- `id` (String) Private network identifier (`pn_…`) to look up. Set from the result for other lookups.
- `name` (String) Name of the private network to look up. Set from the result for other lookups.

### Read-Only

- `created_at` (String) Creation timestamp in ISO 8601 format.
- `is_default` (Boolean) Whether this is the project's default private network.
- `mode` (String) Subnet allocation mode: `auto` or `custom`.
