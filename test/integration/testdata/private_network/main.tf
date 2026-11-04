# Private network integration test: a private network with one subnet, two public NAT-style instances,
# a private-only worker, and a default route through one of the NAT instances.
# Needs a location where private networks are available (default FIN-03).

variable "name_suffix" {
  description = "Makes private network and subnet names unique per test run."
  type        = string
}

variable "private_network_name" {
  type    = string
  default = "itest-pn"
}

variable "subnet_name" {
  type    = string
  default = "itest-app"
}

variable "route_via" {
  description = "Which NAT instance the default route goes through: a or b."
  type        = string
  default     = "a"
}

variable "manage_defaults" {
  description = "Also manage the project's default private network and subnet. Changes the project default; only for dedicated test projects."
  type        = bool
  default     = false
}

# Instances need an SSH key: the API rejects an instance create with none.
resource "verda_ssh_key" "test" {
  name       = "itest-pn-key-${var.name_suffix}"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDRfEpwA9VjHSHBQ3H7pZ5TxGkYYxBN3dZwVS5yLK5sxKCxkXc2pZP5cCzRm1LqHRBMDpBYZJqvC9Wvp4sbKe0n9dHIzF7Cr9kpLPQPz6jjqKLrWfU7gZWy9LrCJPYALO1Yf4ZDEz2Z0dLKz5bXGhYa1Z3E8dJPLZvJV5SH5NtYJF7gN7w7F3hV7cLuVHU5Dh0Z5qK1mYj9Z3GhE5nT2YHkM7F5vJV5SH5NtYJF7gN7w7F3hV7cLuVHU5Dh0Z5qK1mYj9Z3GhE5nT instance-test@verda.cloud"
}

resource "verda_private_network" "test" {
  name = "${var.private_network_name}-${var.name_suffix}"
}

resource "verda_subnet" "app" {
  private_network_id = verda_private_network.test.id
  location           = var.instance_location
  name               = "${var.subnet_name}-${var.name_suffix}"
  cidr               = "10.231.0.0/24"
}

resource "verda_instance" "nat" {
  for_each = toset(["a", "b"])

  instance_type = var.instance_type
  image         = var.instance_image
  hostname      = "itest-nat-${each.key}-${var.name_suffix}"
  description   = "Private network integration test NAT instance"
  location      = var.instance_location
  ssh_key_ids   = [verda_ssh_key.test.id]

  network {
    subnet_id = verda_subnet.app.id
  }
}

resource "verda_instance" "worker" {
  instance_type = var.instance_type
  image         = var.instance_image
  hostname      = "itest-worker-${var.name_suffix}"
  description   = "Private network integration test private worker"
  location      = var.instance_location
  ssh_key_ids   = [verda_ssh_key.test.id]

  network {
    subnet_id  = verda_subnet.app.id
    private_ip = "10.231.0.50"
    public_ip  = "none"
  }
}

resource "verda_network_route" "egress" {
  private_network_id = verda_private_network.test.id
  subnet_id          = verda_subnet.app.id
  destination        = "0.0.0.0/0"
  instance_id        = verda_instance.nat[var.route_via].id
}

resource "verda_private_network_default" "this" {
  count              = var.manage_defaults ? 1 : 0
  private_network_id = verda_private_network.test.id
}

resource "verda_subnet_default" "this" {
  count              = var.manage_defaults ? 1 : 0
  private_network_id = verda_private_network.test.id
  location           = var.instance_location
  subnet_id          = verda_subnet.app.id
}

data "verda_private_network" "by_name" {
  name       = verda_private_network.test.name
  depends_on = [verda_private_network.test]
}

data "verda_subnet" "by_id" {
  private_network_id = verda_private_network.test.id
  id                 = verda_subnet.app.id
  depends_on         = [verda_network_route.egress, verda_instance.worker]
}

output "private_network_id" {
  value = verda_private_network.test.id
}

output "subnet_id" {
  value = verda_subnet.app.id
}

output "route_id" {
  value = verda_network_route.egress.id
}

output "nat_a_id" {
  value = verda_instance.nat["a"].id
}

output "nat_b_id" {
  value = verda_instance.nat["b"].id
}

output "worker_id" {
  value = verda_instance.worker.id
}

output "worker_private_network_id" {
  value = verda_instance.worker.private_network_id
}

output "worker_private_ip" {
  value = verda_instance.worker.private_ip_address
}

output "worker_public_ip" {
  value = verda_instance.worker.ip
}

output "ds_private_network_id" {
  value = data.verda_private_network.by_name.id
}

output "ds_subnet_used_ip_count" {
  value = data.verda_subnet.by_id.used_ip_count
}

output "ds_subnet_default_route_instance_id" {
  value = data.verda_subnet.by_id.default_route_instance_id
}
