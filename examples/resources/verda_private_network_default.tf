resource "verda_private_network" "main" {
  name = "main"
}

# Make this private network the project default. Declare at most one per project.
resource "verda_private_network_default" "this" {
  private_network_id = verda_private_network.main.id
}
