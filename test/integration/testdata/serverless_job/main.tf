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

data "verda_serverless_job" "by_name" {
  name       = verda_serverless_job.test.name
  depends_on = [verda_serverless_job.test]
}

data "verda_serverless_job_scaling" "by_name" {
  name       = verda_serverless_job.test.name
  depends_on = [verda_serverless_job.test]
}

data "verda_serverless_job_status" "by_name" {
  name       = verda_serverless_job.test.name
  depends_on = [verda_serverless_job.test]
}

data "verda_serverless_jobs" "all" {
  depends_on = [verda_serverless_job.test]
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

output "job_data_source_name" {
  value = data.verda_serverless_job.by_name.name
}

output "job_data_source_scaling_max_replica_count" {
  value = data.verda_serverless_job_scaling.by_name.max_replica_count
}

output "job_data_source_status" {
  value = data.verda_serverless_job_status.by_name.status
}

output "job_data_source_list_contains_job" {
  value = contains([for job in data.verda_serverless_jobs.all.jobs : job.name], verda_serverless_job.test.name)
}
