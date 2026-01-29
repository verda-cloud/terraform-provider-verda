# Integration test: Secret resource

resource "verda_secret" "test" {
  name  = "integration-test-secret"
  value = "integration-secret-value"
}

output "secret_name" {
  value = verda_secret.test.name
}

output "secret_type" {
  value = verda_secret.test.secret_type
}
