# Basic instance example
resource "verda_instance" "example" {
  instance_type = "1B200.30V"
  image         = "ubuntu-24.04-cuda-12.8-open-docker"
  hostname      = "my-instance"
  description   = "Example GPU instance"
  location      = "FIN-03"

  ssh_key_ids = [verda_ssh_key.example.id]
}

# Instance with startup script and an existing volume
resource "verda_instance" "with_volume" {
  instance_type     = "1B200.30V"
  image             = "ubuntu-24.04-cuda-12.8-open-docker"
  hostname          = "ml-workstation"
  location          = "FIN-03"
  description       = "ML instance with startup script"
  startup_script_id = verda_startup_script.example.id
  ssh_key_ids       = [verda_ssh_key.example.id]

  existing_volumes = [verda_volume.data.id]
}

# Spot instance with OS volume and data volume discontinuation policies
resource "verda_instance" "spot_with_volumes" {
  instance_type = "1B200.30V"
  image         = "ubuntu-24.04-cuda-12.8-open-docker"
  hostname      = "spot-workstation"
  description   = "Spot GPU instance with volume policies"
  location      = "FIN-03"
  is_spot       = true

  ssh_key_ids = [verda_ssh_key.example.id]

  # OS volume: keep detached when spot is discontinued
  os_volume {
    name              = "os-vol"
    size              = 55
    type              = "NVMe"
    on_spot_discontinue = "keep_detached" # Valid: "keep_detached", "move_to_trash", "delete_permanently"
  }

  # Data volume: move to trash when spot is discontinued
  volumes {
    name              = "data-vol"
    size              = 500
    type              = "NVMe"
    on_spot_discontinue = "move_to_trash" # Valid: "keep_detached", "move_to_trash", "delete_permanently"
  }
}

# Output instance information
output "instance_ip" {
  value = verda_instance.example.ip
}

output "instance_status" {
  value = verda_instance.example.status
}
