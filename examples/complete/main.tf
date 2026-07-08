# Complete example: a full console-server baseline on one Nodegrid device.
#
# nodegrid_settings covers everything expressible as a settings-tree path —
# hostname, DNS, NTP, system preferences, cloud enrollment, network
# connections, serial port labels — with drift detection.
#
# nodegrid_exec covers list-shaped configuration built with add/delete
# (firewall rules, DHCP scopes, NAT chains), which has no settings path.
#
# Nodegrid allows only ONE configuration session per device at a time, so
# resources touching the same device are chained with depends_on.

terraform {
  required_providers {
    nodegrid = {
      source = "rayval1/nodegrid"
    }
  }
}

provider "nodegrid" {
  username = "admin" # password via NODEGRID_PASSWORD
}

variable "host" {
  type    = string
  default = "192.0.2.10"
}

variable "hostname" {
  type    = string
  default = "console-server-01"
}

# ── Identity, DNS, cloud, system preferences ────────────────────────────────
resource "nodegrid_settings" "baseline" {
  host = var.host
  settings = {
    # Network identity
    "/settings/network_settings/hostname"           = var.hostname
    "/settings/network_settings/global_dns_servers" = "192.0.2.53 198.51.100.53"
    "/settings/network_settings/domain_name"        = "example.com"

    # ZPE Cloud enrollment
    "/settings/zpe_cloud/enable_zpe_cloud"     = "yes"
    "/settings/zpe_cloud/enable_remote_access" = "yes"

    # System preferences
    "/settings/system_preferences/address_location"               = "Example DC, 100 Main St"
    "/settings/system_preferences/coordinates"                    = "37.7749,-122.4194"
    "/settings/system_preferences/idle_timeout"                   = "36000"
    "/settings/system_preferences/show_hostname_on_webui_header"  = "yes"
    "/settings/system_preferences/enable_banner"                  = "yes"
    "/settings/system_preferences/banner"                         = "Authorized access only."
  }
}

# ── NTP ─────────────────────────────────────────────────────────────────────
resource "nodegrid_settings" "ntp" {
  host = var.host
  settings = {
    "/settings/date_and_time/ntp_mode"            = "client"
    "/settings/date_and_time/ntp_server_1"        = "192.0.2.123"
    "/settings/date_and_time/ntp_server_1_prefer" = "yes"
    "/settings/date_and_time/ntp_server_2"        = "198.51.100.123"
  }

  depends_on = [nodegrid_settings.baseline] # one config session per device
}

# ── Static management interface ─────────────────────────────────────────────
resource "nodegrid_settings" "eth0" {
  host = var.host
  settings = {
    "/settings/network_connections/ETH0/connect_automatically"      = "yes"
    "/settings/network_connections/ETH0/set_as_primary_connection"  = "yes"
    "/settings/network_connections/ETH0/ipv4_mode"                  = "static"
    "/settings/network_connections/ETH0/ipv4_address"               = var.host
    "/settings/network_connections/ETH0/ipv4_bitmask"               = "24"
    "/settings/network_connections/ETH0/ipv4_gateway"               = "192.0.2.1"
    "/settings/network_connections/ETH0/ipv4_dns_server"            = "192.0.2.53"
    "/settings/network_connections/ETH0/ipv6_mode"                  = "no_ipv6_address"
  }

  depends_on = [nodegrid_settings.ntp]
}

# ── Serial port labels (settings paths under /settings/devices) ─────────────
resource "nodegrid_settings" "serial_labels" {
  host = var.host
  settings = {
    "/settings/devices/ttyS1/access/description" = "Leaf switch 1"
    "/settings/devices/ttyS2/access/description" = "Leaf switch 2"
    "/settings/devices/ttyS3/access/description" = "Spine switch"
    "/settings/devices/ttyS4/access/description" = "Server iLO"
  }

  depends_on = [nodegrid_settings.eth0]
}

# ── List-shaped config: firewall rule + DHCP scope via raw CLI ──────────────
resource "nodegrid_exec" "lan_services" {
  host = var.host
  commands = [
    "cd /settings/ipv4_firewall/chains/INPUT",
    "add",
    "set rule_number=0 target=ACCEPT protocol=tcp destination_port=22 description=allow-ssh",
    "commit",
    "cd /settings/dhcp_server/",
    "add",
    "set protocol=dhcp4 subnet=203.0.113.0 netmask=255.255.255.0 router_ip=203.0.113.1 domain_name_servers=192.0.2.53 lease_time=86400",
    "commit",
  ]

  # Optional: undo on destroy. Omit to leave the device untouched.
  # destroy_commands = [ ... ]

  depends_on = [nodegrid_settings.serial_labels]
}

# ── Inspect live state ──────────────────────────────────────────────────────
data "nodegrid_settings" "verify" {
  host = var.host
  path = "/settings/network_settings"

  depends_on = [nodegrid_settings.baseline]
}

output "live_network_settings" {
  value = data.nodegrid_settings.verify.settings
}
