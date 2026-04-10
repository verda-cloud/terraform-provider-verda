# Integration test: Volume resource
# This test follows the documentation examples exactly

resource "verda_volume" "test" {
  name     = "integration-test-volume"
  size     = 100 # GB
  type     = "NVMe"
  location = "FIN-01"

  # Optional: action on spot instance discontinuation
  # Valid values: "keep_detached", "move_to_trash", "delete_permanently"
  on_spot_discontinue = "keep_detached"
}

# Output volume information for verification
output "volume_id" {
  value = verda_volume.test.id
}

output "volume_status" {
  value = verda_volume.test.status
}

output "volume_name" {
  value = verda_volume.test.name
}
