# Example: manage global DNS + domain on a fleet of Nodegrid devices.
#
# Unlike provisioner-based approaches, `terraform plan` shows real drift:
# the provider reads current device state via export_settings.

terraform {
  required_providers {
    nodegrid = {
      source = "rayval1/nodegrid"
    }
  }
}

provider "nodegrid" {
  username = "admin" # password comes from NODEGRID_PASSWORD
}

variable "devices" {
  type        = map(string)
  description = "Device name to management IP."
  default = {
    "console-server-01" = "192.0.2.10"
    "console-server-02" = "192.0.2.11"
  }
}

variable "dns_servers" {
  type    = list(string)
  default = ["192.0.2.53", "198.51.100.53"]
}

variable "domain_name" {
  type    = string
  default = "example.com"
}

resource "nodegrid_settings" "dns" {
  for_each = var.devices

  host = each.value
  settings = {
    "/settings/network_settings/global_dns_servers" = join(" ", var.dns_servers)
    "/settings/network_settings/domain_name"        = var.domain_name
  }
}

# Read-only inspection of live device state:
# data "nodegrid_settings" "net" {
#   host = var.devices["console-server-01"]
#   path = "/settings/network_settings"
# }
# output "live_network_settings" { value = data.nodegrid_settings.net.settings }
