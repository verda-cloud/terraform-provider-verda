data "verda_private_network" "default" {
  default = true
}

# The default private network's default subnet in FIN-03
data "verda_subnet" "default" {
  private_network_id = data.verda_private_network.default.id
  location           = "FIN-03"
  default            = true
}

# A subnet by name, narrowed to one location
data "verda_subnet" "app" {
  private_network_id = data.verda_private_network.default.id
  name               = "app"
  location           = "FIN-03"
}

output "free_ips" {
  value = data.verda_subnet.default.total_ip_count - data.verda_subnet.default.used_ip_count
}
