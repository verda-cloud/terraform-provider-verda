# Node pools of a managed Kubernetes cluster (experimental API)
#
# A cluster must exist first; pools reference it by its composite id. The
# minimal cluster below is included so this example is runnable on its own —
# a real setup would use an existing cluster or verda_managed_kubernetes_cluster.
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

resource "verda_ssh_key" "k8s_nodes" {
  name       = "k8s-nodes"
  public_key = file("~/.ssh/id_ed25519.pub")
}

# CPU pool for general workloads
resource "verda_managed_kubernetes_node_pool" "cpu_workers" {
  cluster_id = verda_managed_kubernetes_cluster.main.id

  name          = "cpu-workers"
  instance_type = "CPU-TURIN.4V.16G"
  replicas      = 2 # the only mutable attribute; 0 scales the pool down to zero
  ssh_key_ids   = [verda_ssh_key.k8s_nodes.id]

  # Create waits for state "active", delete waits until the pool is gone.
  timeouts = {
    create = "10m"
    update = "10m"
    delete = "10m"
  }
}

# GPU pool for accelerating workloads
resource "verda_managed_kubernetes_node_pool" "gpu_workers" {
  cluster_id = verda_managed_kubernetes_cluster.main.id

  name          = "gpu-workers"
  instance_type = "1B200.30V"
  replicas      = 1
  ssh_key_ids   = [verda_ssh_key.k8s_nodes.id]
}
