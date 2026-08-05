---
page_title: "Common tasks"
subcategory: ""
description: |-
  Worked examples for the configuration people change most often: hostname, DNS, NTP, system preferences, ZPE Cloud, network interfaces, serial ports, and firewall rules.
---

# Common tasks

Nodegrid exposes its whole configuration as a tree of
`/settings/<section>/<field>=<value>` pairs. That means one resource,
[`nodegrid_settings`](../resources/settings.md), covers nearly everything —
you are not limited to the sections shown here. Anything you can reach by
typing `cd` and `set` in the Nodegrid CLI, you can write as a settings path.

Use [`nodegrid_exec`](../resources/exec.md) only for configuration built with
`add`/`delete`/`edit` rather than paths: firewall rules, DHCP scopes, NAT
chains, bonding, and port renames.

To find the path for something not listed here, run `export_settings
/settings/<section>` on the device and read the keys it prints. Those keys are
exactly what goes in the `settings` map.

## Hostname

The simplest possible change:

```terraform
resource "nodegrid_settings" "hostname" {
  host = "192.0.2.10"
  settings = {
    "/settings/network_settings/hostname" = "console-server-01"
  }
}
```

Because reads run `export_settings`, this gives you real drift detection: if
someone renames the device in the web UI, the next `terraform plan` shows the
difference and offers to restore it.

Across a fleet, derive the name from a convention rather than repeating it:

```terraform
variable "racks" {
  type    = map(string) # rack number => management IP
  default = { "101" = "192.0.2.11", "105" = "192.0.2.15" }
}

resource "nodegrid_settings" "hostname" {
  for_each = var.racks

  host = each.value
  settings = {
    "/settings/network_settings/hostname" = "in-bang-dc-1-${each.key}-ce1"
  }
}
```

## DNS and domain

```terraform
resource "nodegrid_settings" "dns" {
  host = "192.0.2.10"
  settings = {
    "/settings/network_settings/global_dns_servers" = "192.0.2.53 198.51.100.53"
    "/settings/network_settings/domain_name"        = "example.com"
  }
}
```

Multiple DNS servers are one space-joined string, not a list — `join(" ",
var.dns_servers)` if you keep them as a list elsewhere.

## NTP

```terraform
resource "nodegrid_settings" "ntp" {
  host = "192.0.2.10"
  settings = {
    "/settings/date_and_time/ntp_mode"          = "enabled"
    "/settings/date_and_time/ntp_server_1"      = "192.0.2.123"
    "/settings/date_and_time/ntp_server_1_prefer" = "yes"
    "/settings/date_and_time/ntp_server_2"      = "198.51.100.123"
  }
}
```

## System preferences and ZPE Cloud

```terraform
resource "nodegrid_settings" "system" {
  host = "192.0.2.10"
  settings = {
    "/settings/system_preferences/idle_timeout"        = "3600"
    "/settings/system_preferences/enable_banner"       = "yes"
    "/settings/zpe_cloud/enable_zpe_cloud"             = "yes"
    "/settings/zpe_cloud/enable_remote_access"         = "yes"
  }
}
```

## Static management address

```terraform
resource "nodegrid_settings" "eth0" {
  host = "192.0.2.10"
  settings = {
    "/settings/network_connections/ETH0/connect_automatically" = "yes"
    "/settings/network_connections/ETH0/ipv4_mode"             = "static"
    "/settings/network_connections/ETH0/ipv4_address"          = "192.0.2.10"
    "/settings/network_connections/ETH0/ipv4_bitmask"          = "24"
    "/settings/network_connections/ETH0/ipv4_gateway"          = "192.0.2.1"
  }
}
```

~> **You can cut yourself off.** An address change commits immediately and the
SSH session you are connected over may drop mid-apply. Re-address the
interface you are *not* connected through, and never split a move of one
address between two interfaces across two resources — an IPv4 address cannot
exist on both at once, so both interfaces must change in a single commit.
Use one `nodegrid_exec` block for that case.

## Serial port labels

Renaming a port changes its key in the settings tree, so it needs
`nodegrid_exec`; the attributes afterwards are ordinary settings paths:

```terraform
resource "nodegrid_exec" "rename" {
  host = "192.0.2.10"
  commands = [
    "cd /settings/devices",
    "edit ttyS1",
    "set name=in-bang-dc-1-105-s1",
    "set mode=enabled",
    "commit",
  ]
}

resource "nodegrid_settings" "port_access" {
  host = "192.0.2.10"
  settings = {
    "/settings/devices/in-bang-dc-1-105-s1/access/description" = "Leaf switch 1"
  }

  depends_on = [nodegrid_exec.rename]
}
```

!> `edit ttyS1` fails once the port has been renamed, because `ttyS1` no
longer exists under `/settings/devices`. The rename batch is not idempotent —
run it on first provisioning and guard re-runs with `count`.

## Firewall rules

```terraform
resource "nodegrid_exec" "allow_ssh" {
  host = "192.0.2.10"
  commands = [
    "cd /settings/ipv4_firewall/chains/INPUT",
    "add",
    "set rule_number=0 target=ACCEPT protocol=tcp destination_port=22 description=allow-ssh",
    "commit",
  ]
}
```

## Ordering

A Nodegrid device permits **one configuration session at a time**. A second
concurrent session fails with *"The system configuration has been changed.
Please revert."* Terraform will happily run independent resources in parallel,
so chain everything targeting the same device with `depends_on`:

```terraform
resource "nodegrid_settings" "hostname" { /* ... */ }

resource "nodegrid_settings" "dns" {
  # ...
  depends_on = [nodegrid_settings.hostname]
}
```

Different devices still apply in parallel — only same-device work needs
serializing.

## Reading state without changing it

The data source runs `export_settings` and nothing else, which makes it safe
to point at production:

```terraform
data "nodegrid_settings" "live" {
  host = "192.0.2.10"
  path = "/settings/network_settings"
}

output "current" {
  value = data.nodegrid_settings.live.settings
}
```

This is the recommended way to try the provider against a new device or
firmware revision for the first time.
