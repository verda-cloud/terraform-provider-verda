# File secret example
resource "verda_file_secret" "example" {
  name = "config-files"

  files = [
    {
      file_name      = "config.json"
      base64_content = base64encode("{\"enabled\":true}")
    }
  ]
}

output "file_secret_name" {
  value = verda_file_secret.example.name
}
