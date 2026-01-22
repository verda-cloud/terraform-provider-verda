---
page_title: "verda_startup_script Resource - Verda Provider"
subcategory: "Compute"
description: |-
  Manages startup scripts that execute when instances are created.
---

# verda_startup_script (Resource)

Manages startup scripts that execute automatically when Verda compute instances are created. Use startup scripts to install software, configure services, or perform other initialization tasks.

## Example Usage

### Basic Script

```terraform
resource "verda_startup_script" "basic" {
  name   = "basic-setup"
  script = <<-EOF
    #!/bin/bash
    apt-get update
    apt-get install -y curl wget git
  EOF
}
```

### Docker Installation

```terraform
resource "verda_startup_script" "docker" {
  name   = "docker-setup"
  script = <<-EOF
    #!/bin/bash
    set -e

    # Update system packages
    apt-get update && apt-get upgrade -y

    # Install Docker
    apt-get install -y docker.io docker-compose

    # Enable and start Docker
    systemctl enable docker
    systemctl start docker

    # Allow non-root Docker access
    usermod -aG docker ubuntu
  EOF
}
```

### ML Environment Setup

```terraform
resource "verda_startup_script" "ml_env" {
  name   = "ml-environment"
  script = <<-EOF
    #!/bin/bash
    set -e

    # Install Python and pip
    apt-get update
    apt-get install -y python3 python3-pip python3-venv

    # Install common ML libraries
    pip3 install numpy pandas scikit-learn torch torchvision

    # Install CUDA toolkit (if not pre-installed)
    # apt-get install -y nvidia-cuda-toolkit

    echo "ML environment ready" >> /var/log/startup.log
  EOF
}
```

### Using with Instances

```terraform
resource "verda_startup_script" "init" {
  name   = "instance-init"
  script = <<-EOF
    #!/bin/bash
    echo "Instance initialized at $(date)" >> /var/log/init.log
  EOF
}

resource "verda_instance" "server" {
  instance_type     = "small"
  image             = "ubuntu-22.04"
  hostname          = "my-instance"
  description       = "Instance with startup script"
  location          = "FIN-01"
  startup_script_id = verda_startup_script.init.id

  ssh_key_ids = [verda_ssh_key.main.id]
}
```

~> **Note:** Startup scripts run as root during instance initialization. Ensure your scripts handle errors appropriately using `set -e` or similar error handling.

## Schema

### Required

- `name` (String) Name of the startup script.
- `script` (String) Script content to execute on instance startup. Must start with a shebang (e.g., `#!/bin/bash`).

### Read-Only

- `created_at` (String) Creation timestamp in ISO 8601 format.
- `id` (String) Unique startup script identifier.

## Import

Existing startup scripts can be imported using the script ID:

```shell
terraform import verda_startup_script.example <startup-script-id>
```
