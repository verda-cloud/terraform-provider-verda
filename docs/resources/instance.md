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

### Instance on a Private Network

Place an instance on a private network subnet with the `network` block. The instance always gets a private IP in the subnet; set `public_ip = "none"` for a private-only instance. The instance's `location` must match the subnet's.

```terraform
resource "verda_private_network" "example" {
  name = "my-network"
}

resource "verda_subnet" "app" {
  private_network_id = verda_private_network.example.id
  location           = "FIN-03"
  name               = "app"
  cidr               = "10.10.0.0/24"
}

resource "verda_instance" "on_private_network" {
  instance_type = "1B200.30V"
  image         = "ubuntu-24.04-cuda-12.8-open-docker"
  hostname      = "private-worker"
  description   = "GPU instance on a private network"
  location      = "FIN-03"

  ssh_key_ids = [verda_ssh_key.example.id]

  network {
    subnet_id  = verda_subnet.app.id
    private_ip = "10.10.0.10"
    public_ip  = "none"
  }
}
```

-> **Note:** Verda Private Networks are rolling out across the Verda fleet: they are available in `FIN-03` today, with more locations coming. The locations API reports `is_private_networks_enabled` for each location.

-> **Tip:** A private-only instance can reach the internet through a NAT instance and a [`verda_network_route`](network_route.md).

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
- `network` (Block) Places the instance on a private network subnet. Leave the block out for public networking without a private network; the instance does not join the project's default private network. Changing anything in the block replaces the instance. See [below for nested schema](#nestedblock--network).
- `os_volume` (Attributes) OS volume configuration. See [below for nested schema](#nestedatt--os_volume).
- `pricing` (String) Pricing model for the instance.
- `ssh_key_ids` (List of String) List of SSH key IDs to add to the instance.
- `startup_script_id` (String) ID of the startup script to run on instance creation.
- `timeouts` (Attributes) `create` (default `20m`): how long the create may take, from placing the order to the instance running with an address, before the apply fails. Changing it later updates state in place without touching the instance. A failed create leaves the instance in state as tainted, so the next apply replaces it; run `terraform untaint` on the resource to keep it. The API accepts the order before the instance exists; the resource returns once its addresses are known (`ip`, and `private_ip_address` for an instance with a `network` block).
- `volumes` (Attributes List) Volumes to create and attach to the instance. See [below for nested schema](#nestedatt--volumes).

### Read-Only

- `cpu` (Attributes) CPU information. See [below for nested schema](#nestedatt--cpu).
- `created_at` (String) Creation timestamp in ISO 8601 format.
- `gpu` (Attributes) GPU information. See [below for nested schema](#nestedatt--gpu).
- `gpu_memory` (Attributes) GPU memory information. See [below for nested schema](#nestedatt--gpu_memory).
- `id` (String) Unique instance identifier.
- `ip` (String) Public IP address of the instance. Null for a private-only instance (`network.public_ip = "none"`).
- `memory` (Attributes) System memory information. See [below for nested schema](#nestedatt--memory).
- `os_name` (String) Operating system name.
- `os_volume_id` (String) ID of the OS volume.
- `price_per_hour` (Number) Price per hour for the instance.
- `private_ip_address` (String) Private IP address of the instance in its private network subnet. Null for an instance without a `network` block.
- `status` (String) Current status of the instance (e.g., `running`, `stopped`).
- `storage` (Attributes) Storage information. See [below for nested schema](#nestedatt--storage).
- `private_network_id` (String) ID of the private network the instance is on. Null for an instance without a `network` block.

<a id="nestedblock--network"></a>

### Nested Schema for `network`

Required:

- `subnet_id` (String) ID of the private network subnet (`sn_…`) to place the instance on. Must be in the instance's location.

Optional:

- `private_ip` (String) `auto` (default) to let the platform pick an address in the subnet, or a specific IPv4 address in the subnet. The assigned address is exported as `private_ip_address`. Set the instance's `location` explicitly to a location where private networks are available: it defaults to `FIN-01`, which private networks have not reached yet.
- `public_ip` (String) `auto` (default) to give the instance a public IP as well, or `none` for a private-only instance. A private-only instance can reach the internet through a [`verda_network_route`](network_route.md).

Private network placement is fixed at deploy time: changing any of these replaces the instance. The one exception is switching `private_ip` between `auto` and the address the instance already has: that only updates state. On import, `private_ip` is set to `auto`, and `public_ip` to `auto` or `none` depending on whether the instance has a public IP; a config that pins the instance's current private IP therefore plans an in-place state update, not a replacement.

<a id="nestedatt--os_volume"></a>

### Nested Schema for `os_volume`

Required:

- `name` (String) Name of the OS volume.
- `size` (Number) Size of the OS volume in GB.
- `type` (String) Type of the OS volume (e.g., `NVMe`).

Optional:

- `on_destroy` (String) What happens to the OS volume when the instance is destroyed: `delete_permanently` (default), `move_to_trash`, or `keep_detached`. Can be changed in place; it is only read at destroy time.
- `on_spot_discontinue` (String) Action to take on spot instance discontinuation: `keep_detached`, `move_to_trash`, or `delete_permanently`.

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
