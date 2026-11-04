# A private network is global; add subnets to it per location with verda_subnet.
resource "verda_private_network" "example" {
  name = "my-network"

  # Delete retries while instances on the private network finish deleting (default 10m).
  timeouts = {
    delete = "15m"
  }
}

output "private_network_id" {
  value = verda_private_network.example.id
}
