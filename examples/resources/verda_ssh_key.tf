# SSH key from a local file
resource "verda_ssh_key" "example" {
  name       = "my-ssh-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

# Output SSH key information
output "ssh_key_id" {
  value = verda_ssh_key.example.id
}

output "ssh_key_fingerprint" {
  value = verda_ssh_key.example.fingerprint
}
