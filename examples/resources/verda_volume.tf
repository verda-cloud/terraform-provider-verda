# Basic volume example
resource "verda_volume" "data" {
  name     = "my-data-volume"
  size     = 100 # GB
  type     = "NVMe"
  location = "FIN-01"
}

# Volume with spot discontinuation policy
resource "verda_volume" "spot_data" {
  name     = "spot-data-volume"
  size     = 500 # GB
  type     = "NVMe"
  location = "FIN-01"

  # Action when the attached spot instance is discontinued
  # Valid values: "keep_detached", "move_to_trash", "delete_permanently"
  on_spot_discontinue = "keep_detached"
}

# Output volume information
output "volume_id" {
  value = verda_volume.data.id
}

output "volume_status" {
  value = verda_volume.data.status
}
