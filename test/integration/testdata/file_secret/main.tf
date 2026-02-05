# Integration test: File secret resource

resource "verda_file_secret" "test" {
  name = "integration-test-file-secret"

  files = [
    {
      file_name      = "tokens.txt"
      base64_content = base64encode("token-123")
    }
  ]
}

output "file_secret_name" {
  value = verda_file_secret.test.name
}

output "file_secret_file_names" {
  value = verda_file_secret.test.file_names
}
