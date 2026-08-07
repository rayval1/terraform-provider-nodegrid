# Devices on a NAT'd LAN reachable only from a router unit.
# See docs/guides/5-devices-behind-nat.md.
terraform {
  required_providers {
    nodegrid = { source = "rayval1/nodegrid", version = "~> 0.2" }
  }
}

provider "nodegrid" { username = "admin" }

locals {
  router = "198.51.100.55"

  devices = {
    "01" = { ip = "192.168.0.2", behind_nat = true }
    "02" = { ip = "192.168.0.3", behind_nat = true }
    "03" = { ip = "192.168.0.4", behind_nat = true }
    "04" = { ip = "198.51.100.55", behind_nat = false }
  }
}

resource "nodegrid_settings" "hostname" {
  for_each = local.devices

  host      = each.value.ip
  jump_host = each.value.behind_nat ? local.router : null

  settings = {
    "/settings/network_settings/hostname" = "console-${each.key}"
  }
}

# Reads work through the tunnel too.
data "nodegrid_settings" "behind_nat" {
  host      = local.devices["01"].ip
  jump_host = local.router
  path      = "/settings/network_settings"

  depends_on = [nodegrid_settings.hostname]
}

output "device01_hostname" {
  value = lookup(
    data.nodegrid_settings.behind_nat.settings,
    "/settings/network_settings/hostname",
    "<not reported>",
  )
}
