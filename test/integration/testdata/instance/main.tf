# Integration test: Instance resource
# This test follows the documentation examples exactly
# Instance type, image, and location are configurable via TF_VAR_* environment variables

# First create an SSH key (required for instance)
resource "verda_ssh_key" "test" {
  name       = "instance-test-key"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDRfEpwA9VjHSHBQ3H7pZ5TxGkYYxBN3dZwVS5yLK5sxKCxkXc2pZP5cCzRm1LqHRBMDpBYZJqvC9Wvp4sbKe0n9dHIzF7Cr9kpLPQPz6jjqKLrWfU7gZWy9LrCJPYALO1Yf4ZDEz2Z0dLKz5bXGhYa1Z3E8dJPLZvJV5SH5NtYJF7gN7w7F3hV7cLuVHU5Dh0Z5qK1mYj9Z3GhE5nT2YHkM7F5vJV5SH5NtYJF7gN7w7F3hV7cLuVHU5Dh0Z5qK1mYj9Z3GhE5nT instance-test@verda.cloud"
}

# Create a basic GPU instance following docs example
# Uses variables from variables.tf (can be overridden via TF_VAR_*)
resource "verda_instance" "test" {
  instance_type = var.instance_type
  image         = var.instance_image
  hostname      = "integration-test-instance"
  description   = "Integration test GPU instance"
  location      = var.instance_location

  ssh_key_ids = [verda_ssh_key.test.id]

  # Optional: OS volume with spot discontinuation policy
  os_volume {
    name                = "test-os-vol"
    size                = 55
    type                = "NVMe"
    on_spot_discontinue = "keep_detached"
  }

  # Optional: Additional volumes with spot discontinuation policy
  volumes {
    name                = "test-data-vol"
    size                = 100
    type                = "NVMe"
    on_spot_discontinue = "move_to_trash"
  }
}

# Output instance information for verification
output "instance_id" {
  value = verda_instance.test.id
}

output "instance_ip" {
  value = verda_instance.test.ip
}

output "instance_status" {
  value = verda_instance.test.status
}

output "instance_type" {
  value = verda_instance.test.instance_type
}
