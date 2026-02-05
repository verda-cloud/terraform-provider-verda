# Basic cluster example
resource "verda_cluster" "example" {
  cluster_type = "16H200"
  image        = "ubuntu-22.04-cuda-12.4-cluster"
  hostname     = "training-cluster"
  description  = "Example cluster"
  location     = "FIN-01"

  ssh_key_ids = [verda_ssh_key.example.id]

  shared_volume = {
    name = "cluster-home"
    size = 30000
  }
}

# Output cluster information
output "cluster_id" {
  value = verda_cluster.example.id
}

output "cluster_status" {
  value = verda_cluster.example.status
}
