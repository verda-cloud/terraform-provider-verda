# Secret example
resource "verda_secret" "example" {
  name  = "api-token"
  value = "super-secret-token"
}

output "secret_name" {
  value = verda_secret.example.name
}
