# Static management addressing and route metrics.
# See docs/guides/4-network-interfaces.md.
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

resource "nodegrid_settings" "eth0" {
  host = var.host
  settings = {
    "/settings/network_connections/ETH0/connect_automatically" = "yes"
    "/settings/network_connections/ETH0/ipv4_mode"             = "static"
    "/settings/network_connections/ETH0/ipv4_address"          = "192.0.2.10"
    "/settings/network_connections/ETH0/ipv4_bitmask"          = "24"
    "/settings/network_connections/ETH0/ipv4_gateway"          = "192.0.2.1"
    "/settings/network_connections/ETH0/ipv4_dns_server"       = "192.0.2.53"
  }
}

# Secondary uplink on DHCP, losing the default-route race on purpose
# (higher metric = lower priority).
resource "nodegrid_settings" "bond0" {
  host = var.host
  settings = {
    "/settings/network_connections/BOND0/ipv4_mode"                       = "dhcp"
    "/settings/network_connections/BOND0/ipv4_default_route_metric"       = "90"
    "/settings/network_connections/BOND0/ipv4_ignore_obtained_dns_server" = "yes"
    "/settings/network_connections/BOND0/connect_automatically"           = "yes"
  }

  # One config session per device at a time.
  depends_on = [nodegrid_settings.eth0]
}
