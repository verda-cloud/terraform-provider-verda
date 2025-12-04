# Configure the Verda Provider
terraform {
  required_providers {
    verda = {
      source = "verda-cloud/verda"
    }
  }
}

# Provider configuration with explicit credentials
provider "verda" {
  client_id     = "your-client-id"
  client_secret = "your-client-secret"
  base_url      = "https://api.verda.com" # Optional, uses default if not specified
}

# Alternative: Configure using environment variables
# Set these environment variables:
# export VERDA_CLIENT_ID="your-client-id"
# export VERDA_CLIENT_SECRET="your-client-secret"
# export VERDA_BASE_URL="https://api.verda.com" # Optional

# provider "verda" {
#   # Credentials will be read from environment variables
# }
