# Basic volume example
resource "verda_volume" "data" {
  name     = "my-data-volume"
  size     = 100 # GB
  type     = "NVMe"
  location = "FIN-01"
}

# Output volume information
output "volume_id" {
  value = verda_volume.data.id
}

output "volume_status" {
  value = verda_volume.data.status
}
