# Basic startup script
resource "verda_startup_script" "example" {
  name = "install-docker"
  script = <<-EOF
    #!/bin/bash
    set -e

    # Update system
    apt-get update
    apt-get upgrade -y

    # Install Docker
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh

    # Add ubuntu user to docker group
    usermod -aG docker ubuntu

    echo "Docker installation completed"
  EOF
}

# Output startup script information
output "startup_script_id" {
  value = verda_startup_script.example.id
}
