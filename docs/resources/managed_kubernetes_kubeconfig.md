---
page_title: "verda_managed_kubernetes_kubeconfig Resource - Verda Provider"
subcategory: "Managed Kubernetes"
description: |-
  Issues a client-certificate kubeconfig for a Verda managed Kubernetes cluster (experimental API).
---

# verda_managed_kubernetes_kubeconfig (Resource)

Issues a client certificate for authenticating to a [`verda_managed_kubernetes_cluster`](managed_kubernetes_cluster) and renders it as a ready-to-use kubeconfig. The API has no update operation for kubeconfigs: any change re-issues the certificate by replacing the resource.

-> **Note:** This resource targets an experimental API and may change in backwards-incompatible ways.

~> **Warning:** Deleting this resource removes the server-side record but does **not** revoke the certificate. Already-downloaded kubeconfigs keep working until `not_after`. The provider does not auto-renew: rotate by tainting the resource and re-applying (or with `terraform apply -replace`).

## Example Usage

```terraform
resource "verda_managed_kubernetes_kubeconfig" "admin" {
  cluster_id = verda_managed_kubernetes_cluster.main.id

  common_name  = "admin"
  groups       = ["system:masters"] # cluster-admin
  display_name = "Terraform admin access"
  duration     = "720h"
}

resource "local_file" "kubeconfig" {
  content         = verda_managed_kubernetes_kubeconfig.admin.content
  filename        = "${path.module}/kubeconfig.yaml"
  file_permission = "0600"
}
```

## Schema

### Required

- `cluster_id` (String) Cluster that owns the kubeconfig, referenced by its composite ID `<location>/<cluster-name>`. Changing it forces replacement.
- `common_name` (String) Certificate Common Name identifying the user; control-plane identities (`system:*`) are rejected by the API. Changing it forces replacement.
- `groups` (List of String) Certificate group memberships; `system:masters` grants cluster-admin access. A kubeconfig without groups authenticates but authorizes nothing. Changing it forces replacement.

### Optional

- `display_name` (String) Human-facing label for the kubeconfig (at most 128 characters). Changing it forces replacement.
- `duration` (String) Certificate lifetime in Go duration syntax (e.g., `"720h"`); minimum `1h`, defaults to 90 days when omitted. Changing it forces replacement.
- `timeouts` (Block) Custom `create` and `delete` timeouts (default `5m` / `10m`). Create waits for status `ready`; delete waits until the record is gone.

### Read-Only

- `id` (String) Server-generated certificate ID.
- `content` (String, Sensitive) Rendered kubeconfig YAML, available once `status` is `ready`.
- `not_after` (String) Certificate expiry timestamp in ISO 8601 format.
- `status` (String) Issuance status: `pending` or `ready`.

## Import

Existing kubeconfigs can be imported using `<location>/<cluster-name>/<kubeconfig-id>` — the cluster coordinates are needed for the API path even though the resource ID is the certificate ID alone:

```shell
terraform import verda_managed_kubernetes_kubeconfig.example FIN-03/my-cluster/kc-abc123
```
