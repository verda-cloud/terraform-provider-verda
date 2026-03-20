# Read one existing serverless job deployment by name
data "verda_serverless_job" "example" {
  name = "existing-job"
}

# Read its scaling configuration
data "verda_serverless_job_scaling" "example" {
  name = data.verda_serverless_job.example.name
}

# Read its runtime status
data "verda_serverless_job_status" "example" {
  name = data.verda_serverless_job.example.name
}

# List all serverless job deployments visible to the account
data "verda_serverless_jobs" "all" {}
