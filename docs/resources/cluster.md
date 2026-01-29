---
page_title: "verda_cluster Resource - Verda Provider"
subcategory: "Clusters"
description: |-
  Manages a Verda cluster deployment.
---

# verda_cluster (Resource)

Manages a Verda cluster deployment. Clusters provide multi-node GPU compute with a shared volume mounted as `/home`.

## Example Usage

### Basic Cluster

```terraform
resource "verda_cluster" "training" {
  cluster_type = "16H200"
  image        = "ubuntu-22.04-cuda-12.4-cluster"
  hostname     = "training-cluster"
  description  = "GPU training cluster"
  location     = "FIN-01"

  ssh_key_ids = [verda_ssh_key.main.id]

  shared_volume = {
    name = "cluster-home"
    size = 30000
  }
}
```

### Cluster with Startup Script and Existing Shared Volumes

```terraform
resource "verda_startup_script" "bootstrap" {
  name   = "cluster-bootstrap"
  script = <<-EOF
    #!/bin/bash
    echo "Hello from Verda"
  EOF
}

resource "verda_cluster" "production" {
  cluster_type     = "16H200"
  image            = "ubuntu-22.04-cuda-12.4-cluster"
  hostname         = "prod-cluster"
  description      = "Production cluster"
  location         = "FIN-01"
  contract         = "PAY_AS_YOU_GO"
  startup_script_id = verda_startup_script.bootstrap.id

  shared_volume = {
    name = "prod-home"
    size = 50000
  }

  existing_volumes = [
    {
      id = "existing-shared-volume-id"
    }
  ]
}
```

## Finding Cluster Types and Images

To discover available cluster types, images, and locations, use the Verda API:

```bash
curl -H "Authorization: Bearer $TOKEN" https://api.verda.com/v1/cluster-types
curl -H "Authorization: Bearer $TOKEN" https://api.verda.com/v1/images/cluster
curl -H "Authorization: Bearer $TOKEN" https://api.verda.com/v1/locations
```

For full API documentation, visit: [Verda API Reference](https://api.verda.com/v1/docs#tag/Clusters)

## Schema

### Required

- `cluster_type` (String) Cluster type (for example, `16H200`).
- `description` (String) Cluster description.
- `hostname` (String) Cluster hostname.
- `image` (String) Cluster image identifier.
- `shared_volume` (Attributes) Shared cluster volume configuration. See [below for nested schema](#nestedatt--shared_volume).

### Optional

- `auto_rental_extension` (Boolean) Enable automatic rental extension for long-term clusters.
- `contract` (String) Contract type for the cluster.
- `existing_volumes` (Attributes List) Existing shared volumes to attach. See [below for nested schema](#nestedatt--existing_volumes).
- `location` (String) Location code (defaults to `FIN-01`).
- `ssh_key_ids` (List of String) SSH key IDs to attach.
- `startup_script_id` (String) Startup script ID to run on cluster initialization.
- `turn_to_pay_as_you_go` (Boolean) Convert to pay-as-you-go after the long-term period ends.

### Read-Only

- `created_at` (String) Creation timestamp.
- `cpu` (Attributes) CPU information. See [below for nested schema](#nestedatt--cpu).
- `gpu` (Attributes) GPU information. See [below for nested schema](#nestedatt--gpu).
- `gpu_memory` (Attributes) GPU memory information. See [below for nested schema](#nestedatt--gpu_memory).
- `id` (String) Cluster identifier.
- `ip` (String) Cluster jump host IP address.
- `long_term_period` (String) Long-term rental period description.
- `memory` (Attributes) Memory information. See [below for nested schema](#nestedatt--memory).
- `os_name` (String) Operating system name.
- `price_per_hour` (Number) Cluster price per hour.
- `shared_volumes` (Attributes List) Shared volumes attached to the cluster. See [below for nested schema](#nestedatt--shared_volumes).
- `status` (String) Cluster status.
- `storage` (Attributes) Storage information. See [below for nested schema](#nestedatt--storage).
- `worker_nodes` (Attributes List) Worker nodes in the cluster. See [below for nested schema](#nestedatt--worker_nodes).

## Nested Schema

### `shared_volume`

Required:

- `name` (String) Shared volume name.
- `size` (Number) Shared volume size in GB.

### `existing_volumes`

Required:

- `id` (String) Existing shared volume ID.

### `cpu`

Read-Only:

- `description` (String)
- `number_of_cores` (Number)

### `gpu`

Read-Only:

- `description` (String)
- `number_of_gpus` (Number)

### `memory`

Read-Only:

- `description` (String)
- `size_in_gigabytes` (Number)

### `gpu_memory`

Read-Only:

- `description` (String)
- `size_in_gigabytes` (Number)

### `storage`

Read-Only:

- `description` (String)

### `shared_volumes`

Read-Only:

- `id` (String)
- `mount_point` (String)
- `name` (String)
- `size_in_gigabytes` (Number)

### `worker_nodes`

Read-Only:

- `id` (String)
- `hostname` (String)
- `private_ip` (String)
- `public_ip` (String)
- `status` (String)
