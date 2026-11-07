---
page_title: "verda_managed_kubernetes_cluster Resource - Verda Provider"
subcategory: "Managed Kubernetes"
description: |-
  Manages a Verda managed Kubernetes cluster (experimental API).
---

# verda_managed_kubernetes_cluster (Resource)

Manages a managed Kubernetes cluster: control plane provisioning, networking, and version upgrades. Node pools are managed separately with the [`verda_managed_kubernetes_node_pool`](managed_kubernetes_node_pool) resource.

-> **Note:** This resource targets an experimental API and may change in backwards-incompatible ways.

## Example Usage

```terraform
# Private networks are not managed by this provider yet: create the network
# and subnet beforehand. The subnet's location must match the cluster
# location and its CIDR must not overlap pod_cidr / service_cidr.
resource "verda_managed_kubernetes_cluster" "main" {
  name               = "my-cluster"
  kubernetes_version = "1.37.0"
  location           = "FIN-03"

  network_id = "pn_xxxxxxxx"
  subnet_id  = "sn_xxxxxxxx"

  pod_cidr     = "10.244.0.0/16"
  service_cidr = "10.96.0.0/16"

  timeouts = {
    create = "10m"
    update = "10m"
    delete = "10m"
  }
}

resource "verda_managed_kubernetes_node_pool" "cpu_workers" {
  cluster_id = verda_managed_kubernetes_cluster.main.id
  # ...
}
```

-> **Tip:** The composite `id` (`<location>/<cluster-name>`) is how node pools and kubeconfigs reference the cluster.

## Schema

### Required

- `name` (String) Name of the cluster; a DNS-1123 label of at most 30 characters. Changing it forces replacement.
- `kubernetes_version` (String) Kubernetes version of the cluster; must exist in the version catalogue. Updating runs a rolling control-plane upgrade in place.
- `location` (String) Location code for the cluster (e.g., `FIN-03`); must match the subnet's location. Changing it forces replacement.
- `network_id` (String) ID of the project's private network. Changing it forces replacement.
- `subnet_id` (String) ID of the subnet within the private network; its CIDR must not overlap `pod_cidr` or `service_cidr`. Changing it forces replacement.
- `pod_cidr` (String) Pod IP range; must be `/21` or larger and must not overlap `service_cidr`. Changing it forces replacement.
- `service_cidr` (String) Service ClusterIP range; must not overlap `pod_cidr`. Changing it forces replacement.

### Optional

- `timeouts` (Block) Custom `create`, `update` and `delete` timeouts (default `30m` each). Create and update wait for phase `running`; delete waits until the cluster is gone.

### Read-Only

- `id` (String) Composite resource ID in the form `<location>/<cluster-name>`.
- `created_at` (String) Creation timestamp in ISO 8601 format.
- `dns_suffix` (String) DNS suffix of the cluster.
- `external_endpoint` (String) External API endpoint of the cluster.
- `phase` (String) Lifecycle phase: `provisioning`, `running`, `degraded` or `deleting`.

## Import

Existing clusters can be imported using the composite ID `<location>/<cluster-name>`:

```shell
terraform import verda_managed_kubernetes_cluster.example FIN-03/my-cluster
```
