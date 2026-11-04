---
page_title: "verda_subnet Resource - Verda Provider"
subcategory: "Networking"
description: |-
  Manages an IPv4 subnet of a Verda private network in one location.
---

# verda_subnet (Resource)

Manages an IPv4 subnet of a Verda private network in one location. Instances are placed on a subnet with the `network` block of [`verda_instance`](instance.md).

-> **Note:** Verda Private Networks are rolling out across the Verda fleet: they are available in `FIN-03` today, with more locations coming. The locations API reports `is_private_networks_enabled` for each location.

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

## CIDR Rules

The provider checks `cidr` at plan time. It must be:

- an IPv4 network address (`10.10.0.0/24`, not `10.10.0.5/24`);
- `/29` or larger;
- inside `10.0.0.0/8`, `172.16.0.0/12` or `192.168.0.0/16`.

Subnets of the same private network must not overlap, in any location. The API rejects an overlapping subnet when it is created.

## Deleting

Deleting a subnet is refused while instances are still on it. Instance deletion finishes in the background, so when you destroy a subnet together with its instances, the delete retries until they are gone, for up to `timeouts.delete` (default `10m`).

## Schema

### Required

- `cidr` (String) IPv4 range of the subnet, e.g. `10.10.0.0/24`. See [CIDR Rules](#cidr-rules). Changing it replaces the subnet.
- `location` (String) Location code, e.g. `FIN-03`. Must be a location where private networks are available. Changing it replaces the subnet.
- `name` (String) Name of the subnet, 1–64 characters, unique within the private network. Can be changed in place.
- `private_network_id` (String) ID of the private network the subnet belongs to. Changing it replaces the subnet.

### Optional

- `timeouts` (Attributes) `delete` (default `10m`): how long the delete keeps retrying while instances on the subnet finish deleting. On timeout the apply fails with the last API error, which names the instances still attached.

### Read-Only

- `created_at` (String) Creation timestamp in ISO 8601 format.
- `id` (String) Unique subnet identifier (`sn_…`).
- `is_default` (Boolean) Whether this is the private network's default subnet in its location. The first subnet created in a location becomes its default automatically. Read-only here; manage it with [`verda_subnet_default`](subnet_default.md).

## Import

Existing subnets can be imported using the private network ID and subnet ID, separated by `/`:

```shell
terraform import verda_subnet.example <private_network_id>/<subnet_id>
```

For example:

```shell
terraform import verda_subnet.example pn_abc123/sn_def456
```
