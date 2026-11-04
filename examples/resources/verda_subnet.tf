resource "verda_private_network" "example" {
  name = "my-network"
}

# Subnets of the same private network must not overlap, in any location.
resource "verda_subnet" "app" {
  private_network_id = verda_private_network.example.id
  location           = "FIN-03"
  name               = "app"
  cidr               = "10.10.0.0/24"
}

output "subnet_id" {
  value = verda_subnet.app.id
}
