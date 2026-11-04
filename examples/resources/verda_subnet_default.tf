resource "verda_private_network" "main" {
  name = "main"
}

resource "verda_subnet" "fin03" {
  private_network_id = verda_private_network.main.id
  location           = "FIN-03"
  name               = "fin03"
  cidr               = "10.20.0.0/24"
}

# Make this subnet the private network's default in FIN-03. Declare at most one per private network and location.
resource "verda_subnet_default" "fin03" {
  private_network_id = verda_private_network.main.id
  location           = "FIN-03"
  subnet_id          = verda_subnet.fin03.id
}
