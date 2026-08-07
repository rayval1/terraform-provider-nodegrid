---
page_title: "5. Devices behind a NAT'd LAN (jump host)"
subcategory: "Examples"
description: |-
  Manage console servers that are only reachable from a router unit, using jump_host to tunnel like ssh -J.
---

# Devices behind a NAT'd LAN

A common topology puts one unit on the routable management network and the
rest on a private LAN behind it, reachable only from that unit. Without a
tunnel those devices are unmanageable — in the fleet this provider was built
for, that was six of eight devices.

Set `jump_host` on the ones that need it. Available on `nodegrid_settings`,
`nodegrid_exec`, and the `nodegrid_settings` data source.

```terraform
data "nodegrid_settings" "behind_nat" {
  host      = "192.168.0.2"      # only reachable from the router unit
  jump_host = "198.51.100.55"    # the routable router unit
  path      = "/settings/network_settings"
}
```

## A mixed fleet

```terraform
locals {
  router = "198.51.100.55"

  racks = {
    "101" = { ip = "192.168.0.2", behind_nat = true }
    "102" = { ip = "192.168.0.3", behind_nat = true }
    "103" = { ip = "192.168.0.4", behind_nat = true }
    "105" = { ip = local.router, behind_nat = false }
  }
}

resource "nodegrid_settings" "hostname" {
  for_each = local.racks

  host      = each.value.ip
  jump_host = each.value.behind_nat ? local.router : null

  settings = {
    "/settings/network_settings/hostname" = "console-${each.key}"
  }
}
```

Pass `null` for directly-reachable devices — the attribute is optional, and a
null value dials straight through.

## How the tunnel works

It is the equivalent of `ssh -J`. A TCP connection is opened to the target
**from** the jump host, then a second SSH handshake runs over that connection.
Authentication to the target is therefore end-to-end, not delegated to the
bastion. The jump host is reached with the same credentials and port as the
target.

## Ordering

If the router unit is itself managed by Terraform, make the devices behind it
depend on it — otherwise a change that drops the router's connectivity can
strand the run halfway through:

```terraform
resource "nodegrid_settings" "router" {
  host     = local.router
  settings = { "/settings/network_settings/hostname" = "console-105" }
}

resource "nodegrid_settings" "behind" {
  for_each = { for k, v in local.racks : k => v if v.behind_nat }

  host      = each.value.ip
  jump_host = local.router
  settings  = { "/settings/network_settings/hostname" = "console-${each.key}" }

  depends_on = [nodegrid_settings.router]
}
```

~> Only one configuration session per device at a time. That limit applies to
the **target**, not the jump host — several devices can tunnel through the
same bastion concurrently.
