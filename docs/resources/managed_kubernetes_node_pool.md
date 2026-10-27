---
page_title: "verda_managed_kubernetes_node_pool Resource - Verda Provider"
subcategory: "Managed Kubernetes"
description: |-
  Manages a node pool of a Verda managed Kubernetes cluster (experimental API).
---

# verda_managed_kubernetes_node_pool (Resource)

Manages a pool of worker nodes for a [`verda_managed_kubernetes_cluster`](managed_kubernetes_cluster). Creating a pool waits until it reports state `active`; scaling `replicas` updates in place.

-> **Note:** This resource targets an experimental API and may change in backwards-incompatible ways.

## Example Usage

```terraform
resource "verda_ssh_key" "k8s_nodes" {
  name       = "k8s-nodes"
  public_key = file("~/.ssh/id_ed25519.pub")
}

resource "verda_managed_kubernetes_node_pool" "cpu_workers" {
  cluster_id = verda_managed_kubernetes_cluster.main.id

  name          = "cpu-workers"
  instance_type = "CPU-TURIN.4V.16G"
  replicas      = 2
  ssh_key_ids   = [verda_ssh_key.k8s_nodes.id]

  timeouts = {
    create = "10m"
    update = "10m"
    delete = "10m"
  }
}
```

-> **Tip:** If a pool stays in `creating` or turns `degraded`, inspect the computed `errors` attribute — it carries machine-readable reasons such as `insufficient_capacity` or `insufficient_funds`.

## Schema

### Required

- `cluster_id` (String) Cluster that owns the pool, referenced by its composite ID `<location>/<cluster-name>` (e.g. `verda_managed_kubernetes_cluster.main.id`). Changing it forces replacement.
- `name` (String) Name of the node pool; a DNS-1123 label of at most 30 characters. Changing it forces replacement.
- `instance_type` (String) Verda VM type for the pool's workers (e.g., `CPU-TURIN.4V.16G`, `1B200.30V`). Changing it forces replacement.
- `ssh_key_ids` (Set of String) SSH keys injected into each worker; at least one required. Changing it forces replacement.

### Optional

- `replicas` (Number) Desired number of worker nodes; defaults to `1`. `0` scales the pool down to zero. This is the only attribute that updates in place.
- `timeouts` (Block) Custom `create`, `update` and `delete` timeouts (default `20m` each). Create and update wait for state `active`; delete waits until the pool is gone.

### Read-Only

- `id` (String) Composite resource ID in the form `<location>/<cluster-name>/<pool-name>`.
- `created_at` (String) Creation timestamp in ISO 8601 format.
- `errors` (List of Object) API-reported reasons for a degraded pool; each entry has `reason`, `message` and `since` attributes.
- `ready_replicas` (Number) Number of workers currently ready.
- `state` (String) Lifecycle state: `creating`, `active`, `degraded` or `deleting`.

## Import

Existing node pools can be imported using the composite ID `<location>/<cluster-name>/<pool-name>`:

```shell
terraform import verda_managed_kubernetes_node_pool.example FIN-03/my-cluster/cpu-workers
```
