---
page_title: "verda_instance Resource - Verda Provider"
subcategory: "Compute"
description: |-
  Manages a Verda GPU compute instance for AI/ML workloads.
---

# verda_instance (Resource)

Manages a Verda GPU compute instance. Instances provide high-performance GPU computing for machine learning training, inference, and other GPU-accelerated workloads.

## Example Usage

### Basic Instance

Create a simple GPU instance with SSH access:

```terraform
resource "verda_instance" "example" {
  instance_type = "1B200.30V"
  image         = "ubuntu-24.04-cuda-12.8-open-docker"
  hostname      = "my-instance"
  description   = "Example GPU instance"
  location      = "FIN-03"

  ssh_key_ids = [verda_ssh_key.example.id]
}
```

### Instance with Startup Script and Volumes

Create an instance with automated setup and attached storage:

```terraform
resource "verda_startup_script" "docker_setup" {
  name   = "docker-setup"
  script = <<-EOF
    #!/bin/bash
    apt-get update && apt-get install -y docker.io
    systemctl enable docker && systemctl start docker
  EOF
}

resource "verda_instance" "production" {
  instance_type = "1B200.30V"
  image         = "ubuntu-24.04-cuda-12.8-open-docker"
  hostname      = "production-server"
  description   = "Production ML training server"
  location      = "FIN-03"

  ssh_key_ids       = [verda_ssh_key.example.id]
  startup_script_id = verda_startup_script.docker_setup.id

  # Create new volumes with the instance
  volumes = [
    {
      name = "training-data"
      size = 500
      type = "NVMe"
    }
  ]

  # Attach pre-existing volumes
  existing_volumes = [verda_volume.models.id]

  # Custom OS volume
  os_volume = {
    name = "os-volume"
    size = 100
    type = "NVMe"
  }
}
```

### Spot Instance

Use spot instances for cost-effective batch processing:

```terraform
resource "verda_instance" "spot" {
  instance_type = "1B200.30V"
  image         = "ubuntu-24.04-cuda-12.8-open-docker"
  hostname      = "batch-processor"
  description   = "Cost-effective batch processing"
  location      = "FIN-03"
  is_spot       = true

  ssh_key_ids = [verda_ssh_key.example.id]
}
```

~> **Note:** Spot instances offer significant cost savings but may be terminated when capacity is needed. Use them for fault-tolerant workloads.

## Finding Available Instance Types and Images

To discover available instance types, images, and locations for your Verda account, use the Verda API:

```bash
# List available instance types
curl -H "Authorization: Bearer $TOKEN" https://api.verda.com/v1/instance-types

# List available images
curl -H "Authorization: Bearer $TOKEN" https://api.verda.com/v1/images

# List available locations
curl -H "Authorization: Bearer $TOKEN" https://api.verda.com/v1/locations
```

For full API documentation, visit: [Verda API Reference](https://api.verda.com/v1/docs#tag/instance-types)

## Schema

### Required

- `description` (String) Description of the instance.
- `hostname` (String) Hostname for the instance.
- `image` (String) Image to use for the instance. Use the API to list available images.
- `instance_type` (String) Type of the instance (e.g., `1B200.30V`). Use the API to list available instance types.

### Optional

- `contract` (String) Contract type for the instance.
- `existing_volumes` (List of String) IDs of existing volumes to attach to the instance.
- `is_spot` (Boolean) Whether this is a spot instance. Defaults to `false`.
- `location` (String) Location code for the instance. Defaults to `FIN-01`.
- `os_volume` (Attributes) OS volume configuration. See [below for nested schema](#nestedatt--os_volume).
- `pricing` (String) Pricing model for the instance.
- `ssh_key_ids` (List of String) List of SSH key IDs to add to the instance.
- `startup_script_id` (String) ID of the startup script to run on instance creation.
- `volumes` (Attributes List) Volumes to create and attach to the instance. See [below for nested schema](#nestedatt--volumes).

### Read-Only

- `cpu` (Attributes) CPU information. See [below for nested schema](#nestedatt--cpu).
- `created_at` (String) Creation timestamp in ISO 8601 format.
- `gpu` (Attributes) GPU information. See [below for nested schema](#nestedatt--gpu).
- `gpu_memory` (Attributes) GPU memory information. See [below for nested schema](#nestedatt--gpu_memory).
- `id` (String) Unique instance identifier.
- `ip` (String) Public IP address of the instance.
- `memory` (Attributes) System memory information. See [below for nested schema](#nestedatt--memory).
- `os_name` (String) Operating system name.
- `os_volume_id` (String) ID of the OS volume.
- `price_per_hour` (Number) Price per hour for the instance.
- `status` (String) Current status of the instance (e.g., `running`, `stopped`).
- `storage` (Attributes) Storage information. See [below for nested schema](#nestedatt--storage).

<a id="nestedatt--os_volume"></a>

### Nested Schema for `os_volume`

Required:

- `name` (String) Name of the OS volume.
- `size` (Number) Size of the OS volume in GB.
- `type` (String) Type of the OS volume (e.g., `NVMe`).

<a id="nestedatt--volumes"></a>

### Nested Schema for `volumes`

Required:

- `name` (String) Name of the volume.
- `size` (Number) Size of the volume in GB.
- `type` (String) Type of the volume (e.g., `NVMe`).

Optional:

- `location` (String) Location code for the volume.

<a id="nestedatt--cpu"></a>

### Nested Schema for `cpu`

Read-Only:

- `description` (String) CPU model description.
- `number_of_cores` (Number) Number of CPU cores.

<a id="nestedatt--gpu"></a>

### Nested Schema for `gpu`

Read-Only:

- `description` (String) GPU model description.
- `number_of_gpus` (Number) Number of GPUs.

<a id="nestedatt--gpu_memory"></a>

### Nested Schema for `gpu_memory`

Read-Only:

- `description` (String) GPU memory description.
- `size_in_gigabytes` (Number) GPU memory size in GB.

<a id="nestedatt--memory"></a>

### Nested Schema for `memory`

Read-Only:

- `description` (String) Memory description.
- `size_in_gigabytes` (Number) Memory size in GB.

<a id="nestedatt--storage"></a>

### Nested Schema for `storage`

Read-Only:

- `description` (String) Storage description.

## Import

Existing instances can be imported using the instance ID:

```shell
terraform import verda_instance.example <instance-id>
```
