---
page_title: "verda_subnet Data Source - Verda Provider"
subcategory: "Networking"
description: |-
  Looks up a subnet of a Verda private network by ID, by name, or as the private network's default subnet in a location.
---

# verda_subnet (Data Source)

Looks up a subnet of a Verda private network by ID, by name, or as the private network's default subnet in a location. Set `private_network_id` and exactly one of `id`, `name` or `default = true`. `default` also needs `location`; with `id` or `name`, `location` narrows the search.

-> **Note:** Verda Private Networks are rolling out across the Verda fleet: they are available in `FIN-03` today, with more locations coming. The locations API reports `is_private_networks_enabled` for each location.

## Example Usage

### Default Subnet of the Default Private Network

```terraform
data "verda_private_network" "default" {
  default = true
}

data "verda_subnet" "default" {
  private_network_id = data.verda_private_network.default.id
  location           = "FIN-03"
  default            = true
}

output "free_ips" {
  value = data.verda_subnet.default.total_ip_count - data.verda_subnet.default.used_ip_count
}
```

### By Name

```terraform
data "verda_subnet" "app" {
  private_network_id = data.verda_private_network.default.id
  name               = "app"
  location           = "FIN-03"
}
```

The lookup fails unless exactly one subnet matches.

## Schema

### Required

- `private_network_id` (String) ID of the private network to search.

### Optional

- `default` (Boolean) Set to `true` to look up the private network's default subnet in `location`. Requires `location`.
- `id` (String) Subnet identifier (`sn_…`) to look up. Set from the result for other lookups.
- `location` (String) Location code, e.g. `FIN-03`. Required with `default`; with `id` or `name` it narrows the search.
- `name` (String) Name of the subnet to look up. Set from the result for other lookups.

### Read-Only

- `cidr` (String) IPv4 range of the subnet.
- `created_at` (String) Creation timestamp in ISO 8601 format.
- `default_route_instance_id` (String) ID of the instance the subnet's default route (`0.0.0.0/0`) points at, if any.
- `is_default` (Boolean) Whether this is the private network's default subnet in its location.
- `total_ip_count` (Number) Number of usable addresses.
- `used_ip_count` (Number) Number of addresses in use.
