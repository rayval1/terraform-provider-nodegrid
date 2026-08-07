# Example: set the hostname on one device, or on a whole fleet.
#
# Hostname is a plain settings path, so `nodegrid_settings` handles it and
# `terraform plan` shows real drift — if someone renames a device from the
# web UI, the next plan shows the change and offers to put it back.

terraform {
  required_providers {
    nodegrid = {
      source  = "rayval1/nodegrid"
      version = "~> 0.1"
    }
  }
}

provider "nodegrid" {
  username = "admin" # password comes from NODEGRID_PASSWORD
}

# --- Single device ----------------------------------------------------------

resource "nodegrid_settings" "hostname" {
  host = "192.0.2.10"
  settings = {
    "/settings/network_settings/hostname" = "console-server-01"
  }
}

# --- A fleet, with a naming convention --------------------------------------
#
# Devices keyed by a short id. The hostname is derived rather than written out
# per device, so the convention lives in one place.

variable "devices" {
  type        = map(string)
  description = "Device key => management IP."
  default = {
    "01" = "192.0.2.11"
    "02" = "192.0.2.12"
    "03" = "192.0.2.15"
  }
}

variable "site" {
  type        = string
  description = "Site code used in the hostname."
  default     = "site1"
}

variable "primary_device" {
  type        = string
  description = "The device acting as router; it gets the primary role suffix."
  default     = "03"
}

resource "nodegrid_settings" "fleet_hostname" {
  for_each = var.devices

  host = each.value
  settings = {
    "/settings/network_settings/hostname" = format(
      "%s-console-%s-%s",
      var.site,
      each.key,
      each.key == var.primary_device ? "primary" : "secondary",
    )
  }
}

# --- Verifying the result ---------------------------------------------------
#
# Reads the subtree back off the device. Useful as a post-apply sanity check,
# or to pull the current value into other configuration.

data "nodegrid_settings" "check" {
  host = var.devices[var.primary_device]
  path = "/settings/network_settings"

  depends_on = [nodegrid_settings.fleet_hostname]
}

output "hostname_on_device" {
  value = lookup(
    data.nodegrid_settings.check.settings,
    "/settings/network_settings/hostname",
    "<not reported by export_settings>",
  )
}

# Note: changing a hostname does not re-key anything else in the settings tree,
# but it DOES change serial-port paths if your ports are named after the host
# (see the serial-ports example). Apply hostname changes before port labels.
