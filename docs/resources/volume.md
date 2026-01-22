---
page_title: "verda_volume Resource - Verda Provider"
subcategory: "Storage"
description: |-
  Manages persistent NVMe storage volumes for Verda instances.
---

# verda_volume (Resource)

Manages persistent NVMe storage volumes that can be attached to Verda compute instances. Volumes persist independently of instances, making them ideal for storing data that needs to survive instance termination.

## Example Usage

### Basic Volume

```terraform
resource "verda_volume" "data" {
  name     = "data-volume"
  size     = 100
  type     = "NVMe"
  location = "FIN-01"
}
```

### Attach Volume to Instance

```terraform
resource "verda_volume" "training_data" {
  name     = "ml-training-data"
  size     = 500
  type     = "NVMe"
  location = "FIN-01"
}

resource "verda_instance" "ml_server" {
  instance_type    = "medium"
  image            = "ubuntu-22.04"
  hostname         = "ml-server"
  description      = "Machine learning server"
  location         = "FIN-01"
  existing_volumes = [verda_volume.training_data.id]

  ssh_key_ids = [verda_ssh_key.main.id]
}
```

### Multiple Volumes

```terraform
resource "verda_volume" "datasets" {
  name     = "datasets"
  size     = 1000
  type     = "NVMe"
  location = "FIN-01"
}

resource "verda_volume" "checkpoints" {
  name     = "model-checkpoints"
  size     = 200
  type     = "NVMe"
  location = "FIN-01"
}

resource "verda_instance" "trainer" {
  instance_type    = "large"
  image            = "ubuntu-22.04"
  hostname         = "trainer"
  description      = "Model training instance"
  location         = "FIN-01"
  existing_volumes = [
    verda_volume.datasets.id,
    verda_volume.checkpoints.id
  ]

  ssh_key_ids = [verda_ssh_key.main.id]
}
```

-> **Tip:** Volumes must be in the same location as the instance they are attached to.

## Schema

### Required

- `name` (String) Name of the volume.
- `size` (Number) Size of the volume in GB.
- `type` (String) Type of the volume. Currently supported: `NVMe`.

### Optional

- `location` (String) Location code for the volume. Defaults to `FIN-01`.

### Read-Only

- `created_at` (String) Creation timestamp in ISO 8601 format.
- `id` (String) Unique volume identifier.
- `instance_id` (String) ID of the instance this volume is attached to, if any.
- `status` (String) Current status of the volume (e.g., `available`, `attached`).

## Import

Existing volumes can be imported using the volume ID:

```shell
terraform import verda_volume.example <volume-id>
```
