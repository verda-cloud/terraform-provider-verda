# Integration test: Container resource
# This test follows the documentation examples exactly

# Basic container deployment using public registry (no credentials needed)
resource "verda_container" "test" {
  name = "integration-test-container"

  compute = {
    name = "H100"
    size = 1
  }

  scaling = {
    min_replica_count               = 1
    max_replica_count               = 5
    queue_message_ttl_seconds       = 3600
    deadline_seconds                = 3600
    concurrent_requests_per_replica = 10

    scale_down_policy = {
      delay_seconds = 300
    }

    scale_up_policy = {
      delay_seconds = 300
    }

    queue_load = {
      threshold = 5.0
    }
  }

  containers = [
    {
      image        = "nginx:1.29.1"
      exposed_port = 80
    }
  ]
}

# Output container information for verification
output "container_name" {
  value = verda_container.test.name
}

output "container_endpoint_base_url" {
  value = verda_container.test.endpoint_base_url
}

output "container_created_at" {
  value = verda_container.test.created_at
}
