# Managed Kubernetes cluster (experimental API)
variable "network_id" {
  description = "ID of the project's private network"
  type        = string
}

variable "subnet_id" {
  description = "ID of the subnet within the private network"
  type        = string
}

resource "verda_managed_kubernetes_cluster" "main" {
  name               = "my-cluster"
  kubernetes_version = "1.37.0"
  location           = "FIN-03"

  network_id = var.network_id
  subnet_id  = var.subnet_id

  pod_cidr     = "10.244.0.0/16" # must be /21 or larger
  service_cidr = "10.96.0.0/16"  # must not overlap pod_cidr

  # Create/update wait for phase "running", delete waits until the cluster
  # is gone.
  timeouts = {
    create = "10m"
    update = "10m"
    delete = "10m"
  }
}

output "external_endpoint" {
  value = verda_managed_kubernetes_cluster.main.external_endpoint
}
