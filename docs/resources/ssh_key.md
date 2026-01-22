---
page_title: "verda_ssh_key Resource - Verda Provider"
subcategory: "Compute"
description: |-
  Manages SSH keys for accessing Verda compute instances.
---

# verda_ssh_key (Resource)

Manages SSH keys for secure access to Verda compute instances. SSH keys must be created before provisioning instances that require SSH access.

## Example Usage

### Using a Local Key File

```terraform
resource "verda_ssh_key" "main" {
  name       = "my-ssh-key"
  public_key = file("~/.ssh/id_rsa.pub")
}
```

### Using an Inline Key

```terraform
resource "verda_ssh_key" "deploy" {
  name       = "deployment-key"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB... user@example.com"
}
```

### Using with Instances

```terraform
resource "verda_ssh_key" "team" {
  name       = "team-key"
  public_key = file("~/.ssh/id_ed25519.pub")
}

resource "verda_instance" "server" {
  instance_type = "small"
  image         = "ubuntu-22.04"
  hostname      = "app-server"
  description   = "Application server"
  location      = "FIN-01"

  ssh_key_ids = [verda_ssh_key.team.id]
}
```

-> **Tip:** You can attach multiple SSH keys to an instance for team access by passing a list of key IDs.

## Schema

### Required

- `name` (String) Name of the SSH key. Used for identification in the Verda console.
- `public_key` (String) Public key content in OpenSSH format.

### Read-Only

- `created_at` (String) Creation timestamp in ISO 8601 format.
- `fingerprint` (String) SSH key fingerprint for verification.
- `id` (String) Unique SSH key identifier.

## Import

Existing SSH keys can be imported using the key ID:

```shell
terraform import verda_ssh_key.example <ssh-key-id>
```
