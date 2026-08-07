# Example: label serial ports.
#
# This one needs BOTH resources, and the reason is worth understanding.
#
# Renaming a port is `edit ttyS1` + `set name=...`, which changes the port's
# own key in the settings tree. That is not a `/settings/<path> = <value>`
# write, so `nodegrid_settings` cannot express it — `nodegrid_exec` does the
# rename, and `nodegrid_settings` manages the attributes afterwards.
#
# IMPORTANT: `edit ttyS1` fails once the port has already been renamed, because
# `ttyS1` no longer exists under /settings/devices. The exec block below is
# therefore NOT idempotent against an already-labeled device. Run it on first
# provisioning; guard re-runs with the `already_labeled` variable.

terraform {
  required_providers {
    nodegrid = {
      source  = "rayval1/nodegrid"
      version = "~> 0.1"
    }
  }
}

provider "nodegrid" {
  username = "admin"
}

variable "host" {
  type    = string
  default = "192.0.2.10"
}

variable "site" {
  type    = string
  default = "site1"
}

variable "device_id" {
  type    = string
  default = "03"
}

variable "already_labeled" {
  type        = bool
  description = "Set true once ports have been renamed, to skip the rename batch."
  default     = false
}

locals {
  prefix = "${var.site}-console-${var.device_id}"

  # Physical port to the role it carries.
  ports = {
    ttyS1 = "sw1"
    ttyS2 = "sw2"
    ttyS3 = "sw3"
    ttyS4 = "bmc"
  }

  # Post-rename names, e.g. console-server-01-sw3.
  named = { for tty, role in local.ports : tty => "${local.prefix}-${role}" }

  descriptions = {
    sw1 = "Switch 1 - device ${var.device_id}"
    sw2 = "Switch 2 - device ${var.device_id}"
    sw3 = "Switch 3 - device ${var.device_id}"
    bmc = "Server BMC - device ${var.device_id}"
  }
}

# --- Step 1: rename the ports (exec — not expressible as settings paths) -----

resource "nodegrid_exec" "rename_ports" {
  count = var.already_labeled ? 0 : 1

  host = var.host
  commands = flatten([
    for tty, name in local.named : [
      "cd /settings/devices",
      "edit ${tty}",
      "set name=${name}",
      "set mode=enabled",
      "commit",
    ]
  ])
}

# --- Step 2: manage attributes by their new paths (settings — gets drift) ----
#
# Only one config session per device at a time, so this must not overlap with
# the rename batch above.

resource "nodegrid_settings" "port_access" {
  host = var.host

  settings = merge([
    for tty, name in local.named : {
      "/settings/devices/${name}/access/description"               = local.descriptions[local.ports[tty]]
      "/settings/devices/${name}/access/enable_hostname_detection" = "no"
    }
  ]...)

  depends_on = [nodegrid_exec.rename_ports]
}

# --- Verify -----------------------------------------------------------------

data "nodegrid_settings" "ports" {
  host = var.host
  path = "/settings/devices"

  depends_on = [nodegrid_settings.port_access]
}

output "device_tree" {
  value = data.nodegrid_settings.ports.settings
}
