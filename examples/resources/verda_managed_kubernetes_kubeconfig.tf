# Client-certificate kubeconfig for a managed Kubernetes cluster
# (experimental API)
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

  pod_cidr     = "10.244.0.0/16"
  service_cidr = "10.96.0.0/16"
}

resource "verda_managed_kubernetes_kubeconfig" "admin" {
  cluster_id = verda_managed_kubernetes_cluster.main.id

  common_name  = "admin"
  groups       = ["system:masters"] # cluster-admin
  display_name = "Terraform admin access"
  duration     = "720h" # 30 days; minimum 1h, default 90 days when omitted
}

# Write the kubeconfig to a file to use with kubectl
resource "local_file" "kubeconfig" {
  content         = verda_managed_kubernetes_kubeconfig.admin.content
  filename        = "${path.module}/kubeconfig.yaml"
  file_permission = "0600"
}

output "kubeconfig" {
  value     = verda_managed_kubernetes_kubeconfig.admin.content
  sensitive = true
}

output "kubeconfig_expires_at" {
  value = verda_managed_kubernetes_kubeconfig.admin.not_after
}
