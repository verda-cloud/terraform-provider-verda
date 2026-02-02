# Integration test: Serverless Job resource
# This test follows the documentation examples exactly

# Basic serverless job deployment using public registry
resource "verda_serverless_job" "test" {
  name = "integration-test-job"

  compute = {
    name = "H100"
    size = 1
  }

  scaling = {
    max_replica_count         = 5
    queue_message_ttl_seconds = 7200
    deadline_seconds          = 3600
  }

  containers = [
    {
      image        = "python:3.11-slim"
      exposed_port = 8080
    }
  ]
}

# Output serverless job information for verification
output "job_name" {
  value = verda_serverless_job.test.name
}

output "job_endpoint_base_url" {
  value = verda_serverless_job.test.endpoint_base_url
}

output "job_created_at" {
  value = verda_serverless_job.test.created_at
}
