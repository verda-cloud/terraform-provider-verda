---
page_title: "verda_network_route Resource - Verda Provider"
subcategory: "Networking"
description: |-
  Routes a private network subnet's traffic for a destination through an instance.
---

# verda_network_route (Resource)

Routes a private network subnet's traffic for a destination through an instance. The typical use is a NAT instance that gives private-only instances on the subnet internet access.

-> **Note:** Verda Private Networks are rolling out across the Verda fleet: they are available in `FIN-03` today, with more locations coming. The locations API reports `is_private_networks_enabled` for each location.

## Example Usage

### NAT Instance

A NAT instance with a public IP, a private-only worker, and a route that sends the subnet's internet-bound traffic through the NAT instance:

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

# The route only steers traffic: the NAT instance must forward and masquerade it.
resource "verda_startup_script" "nat" {
  name   = "nat"
  script = <<-EOF
    #!/bin/bash
    sysctl -w net.ipv4.ip_forward=1
    WAN_IF=$(ip route show default | awk '{print $5; exit}')
    iptables -t nat -A POSTROUTING -s ${verda_subnet.app.cidr} -o "$WAN_IF" -j MASQUERADE
  EOF
}

resource "verda_instance" "nat" {
  instance_type     = "CPU.8V.32G"
  image             = "ubuntu-24.04"
  hostname          = "nat"
  description       = "NAT instance"
  location          = "FIN-03"
  ssh_key_ids       = [verda_ssh_key.example.id]
  startup_script_id = verda_startup_script.nat.id

  network {
    subnet_id = verda_subnet.app.id
    public_ip = "auto"
  }
}

resource "verda_instance" "worker" {
  instance_type = "1B200.30V"
  image         = "ubuntu-24.04-cuda-12.8-open-docker"
  hostname      = "worker"
  description   = "Private-only worker"
  location      = "FIN-03"
  ssh_key_ids   = [verda_ssh_key.example.id]

  network {
    subnet_id = verda_subnet.app.id
    public_ip = "none"
  }
}

# Send the subnet's internet-bound traffic through the NAT instance.
resource "verda_network_route" "egress" {
  private_network_id = verda_private_network.example.id
  subnet_id          = verda_subnet.app.id
  destination        = "0.0.0.0/0"
  instance_id        = verda_instance.nat.id
}
```

~> **Note:** The route only steers traffic to the instance. The NAT instance must itself be configured to forward and masquerade it, for example with a startup script as above. Adapt the script to your image.

## Behavior

- A subnet has at most one route per destination. If the subnet already has a route for `destination` (for example one made in the console), creating this resource takes it over and points it at `instance_id`.
- Changing `instance_id` updates the route in place.
- The instance must be on the same private network, in the subnet's location, and have a public IP.
- The API currently accepts only `0.0.0.0/0` as `destination`.

## Schema

### Required

- `destination` (String) Destination IPv4 CIDR block. Currently the API accepts only `0.0.0.0/0`. Changing it replaces the route.
- `instance_id` (String) ID of the instance to route through. It must be on the same private network, in the subnet's location, and have a public IP. Can be changed in place.
- `subnet_id` (String) ID of the subnet whose traffic is routed. Changing it replaces the route.
- `private_network_id` (String) ID of the private network. Changing it replaces the route.

### Read-Only

- `created_at` (String) Creation timestamp in ISO 8601 format.
- `id` (String) Unique route identifier (`rt_…`).

## Import

Existing routes can be imported using the private network ID, subnet ID and route ID, separated by `/`:

```shell
terraform import verda_network_route.example <private_network_id>/<subnet_id>/<route_id>
```

For example:

```shell
terraform import verda_network_route.example pn_abc123/sn_def456/rt_ghi789
```
