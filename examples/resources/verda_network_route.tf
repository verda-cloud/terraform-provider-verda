# A NAT instance gives private-only instances on the subnet internet access.
resource "verda_private_network" "example" {
  name = "my-network"
}

resource "verda_subnet" "app" {
  private_network_id = verda_private_network.example.id
  location           = "FIN-03"
  name               = "app"
  cidr               = "10.10.0.0/24"
}

# The route only steers traffic: the NAT instance must forward and masquerade it.
resource "verda_startup_script" "nat" {
  name   = "nat"
  script = <<-EOF
    #!/bin/bash
    sysctl -w net.ipv4.ip_forward=1
    WAN_IF=$(ip route show default | awk '{print $5; exit}')
    iptables -t nat -A POSTROUTING -s ${verda_subnet.app.cidr} -o "$WAN_IF" -j MASQUERADE
  EOF
}

resource "verda_instance" "nat" {
  instance_type     = "CPU.8V.32G"
  image             = "ubuntu-24.04"
  hostname          = "nat"
  description       = "NAT instance"
  location          = "FIN-03"
  ssh_key_ids       = [verda_ssh_key.example.id]
  startup_script_id = verda_startup_script.nat.id

  network {
    subnet_id = verda_subnet.app.id
    public_ip = "auto"
  }
}

resource "verda_instance" "worker" {
  instance_type = "1B200.30V"
  image         = "ubuntu-24.04-cuda-12.8-open-docker"
  hostname      = "worker"
  description   = "Private-only worker"
  location      = "FIN-03"
  ssh_key_ids   = [verda_ssh_key.example.id]

  network {
    subnet_id = verda_subnet.app.id
    public_ip = "none"
  }
}

# Send the subnet's internet-bound traffic through the NAT instance.
resource "verda_network_route" "egress" {
  private_network_id = verda_private_network.example.id
  subnet_id          = verda_subnet.app.id
  destination        = "0.0.0.0/0"
  instance_id        = verda_instance.nat.id
}
