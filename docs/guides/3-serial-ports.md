---
page_title: "3. Serial ports: naming and access settings"
subcategory: "Examples"
description: |-
  Label ttyS ports after the equipment they reach, then manage baud rate, protocols and descriptions — using both resources together.
---

# Serial ports: naming and access settings

This is the clearest case where both resources are needed, and understanding
why explains the boundary between them.

Field names on this page are verified against Nodegrid firmware; a real
device reports ~35 fields per port under `/settings/devices/<name>/`.

## Why two resources

Renaming a port is `edit ttyS1` then `set name=...`. That changes the port's
**key** in the settings tree — it is not a `/settings/<path> = <value>` write,
so `nodegrid_settings` cannot express it. `nodegrid_exec` does the rename;
`nodegrid_settings` manages everything afterwards, with drift detection.

## Step 1 — rename

```terraform
locals {
  prefix = "in-bang-dc-1-105"
  ports = {
    ttyS1 = "s1"
    ttyS2 = "s2"
    ttyS3 = "spine"
    ttyS4 = "ilo1"
  }
  named = { for tty, role in local.ports : tty => "${local.prefix}-${role}" }
}

resource "nodegrid_exec" "rename_ports" {
  count = var.already_labeled ? 0 : 1

  host = "192.0.2.10"
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
```

!> **Not idempotent.** `edit ttyS1` fails once the port has been renamed,
because `ttyS1` no longer exists under `/settings/devices`. Run it on first
provisioning and guard re-runs with `count`, as above.

## Step 2 — manage the port's settings

Once renamed, everything about the port is an ordinary settings path:

```terraform
resource "nodegrid_settings" "port_access" {
  host = "192.0.2.10"

  settings = merge([
    for tty, name in local.named : {
      "/settings/devices/${name}/access/description"               = "Uplink ${tty}"
      "/settings/devices/${name}/access/mode"                      = "enabled"
      "/settings/devices/${name}/access/baud_rate"                 = "9600"
      "/settings/devices/${name}/access/data_bits"                 = "8"
      "/settings/devices/${name}/access/stop_bits"                 = "1"
      "/settings/devices/${name}/access/parity"                    = "None"
      "/settings/devices/${name}/access/flow_control"              = "None"
      "/settings/devices/${name}/access/allow_telnet_protocol"     = "no"
      "/settings/devices/${name}/access/allow_ssh_protocol"        = "yes"
      "/settings/devices/${name}/access/multisession"              = "yes"
      "/settings/devices/${name}/access/enable_hostname_detection" = "no"
    }
  ]...)

  depends_on = [nodegrid_exec.rename_ports]
}
```

## Other fields available per port

`escape_sequence`, `power_control_key`, `telnet_port`, `icon`,
`skip_authentication_to_access_device`, `read-write_multisession`,
`allow_binary_socket`, `allow_pre-shared_ssh_key`, `enable_ip_alias`,
`rs-232_signal_for_device_state_detection`, plus `commands/…` and `logging/…`
subtrees for data logging.

~> Port passwords come back from `export_settings` masked as `********`. The
provider keeps the last known value for such fields rather than reporting
drift on every plan.

## Inspecting what exists

```terraform
data "nodegrid_settings" "ports" {
  host = "192.0.2.10"
  path = "/settings/devices"
}
```
