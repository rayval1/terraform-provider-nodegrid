# Firewall, DHCP and NAT — the add/delete-shaped config that needs exec.
# See docs/guides/6-firewall-dhcp-nat.md.
#
# NOTE: these batches APPEND. Re-running duplicates rules.
terraform {
  required_providers {
    nodegrid = { source = "rayval1/nodegrid", version = "~> 0.2" }
  }
}

provider "nodegrid" { username = "admin" }

variable "host" {
  type    = string
  default = "192.0.2.10"
}

variable "lan_cidr" {
  type    = string
  default = "192.168.0.0/24"
}

resource "nodegrid_exec" "firewall" {
  host = var.host
  commands = [
    "cd /settings/ipv4_firewall/chains/INPUT",
    "add",
    "set target=ACCEPT input_interface=lo description=Loopback",
    "add",
    "set target=ACCEPT protocol=icmp description=ICMP",
    "add",
    "set target=ACCEPT protocol=tcp destination_port=22 enable_state_match=yes new=yes description=SSH",
    "add",
    "set target=ACCEPT protocol=tcp destination_port=443 enable_state_match=yes new=yes description=HTTPS",
    "add",
    "set target=ACCEPT enable_state_match=yes related=yes established=yes description=Established",
    "commit",
    "set /settings/ipv4_firewall/policy INPUT=DROP",
    "commit",
  ]
}

resource "nodegrid_exec" "nat" {
  host = var.host
  commands = [
    "set /settings/network_settings enable_ipv4_ip_forward=yes",
    "commit",
    "cd /settings/ipv4_nat/chains/POSTROUTING",
    "add",
    "set target=MASQUERADE output_interface=eth0 source_net4=${var.lan_cidr}",
    "commit",
  ]

  destroy_commands = [
    "cd /settings/ipv4_nat/chains/POSTROUTING",
    "delete 0",
    "commit",
  ]

  depends_on = [nodegrid_exec.firewall]
}
