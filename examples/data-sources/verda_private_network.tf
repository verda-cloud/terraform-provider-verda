# The project's default private network, for example the one the platform created.
data "verda_private_network" "default" {
  default = true
}

# A private network by name
data "verda_private_network" "prod" {
  name = "prod"
}

output "default_private_network_id" {
  value = data.verda_private_network.default.id
}
