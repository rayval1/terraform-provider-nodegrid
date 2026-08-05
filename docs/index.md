---
page_title: "Provider: Nodegrid"
description: |-
  Manage ZPE Nodegrid console servers over SSH using the Nodegrid CLI settings tree.
---

# Nodegrid Provider

The Nodegrid provider manages [ZPE Nodegrid](https://zpesystems.com/) devices
(console servers, services routers) over SSH. It drives the Nodegrid CLI:
reads use `export_settings` (giving real drift detection), writes use
`set` + `commit`.

Community project, not affiliated with ZPE Systems, Inc.

## Example Usage

```terraform
provider "nodegrid" {
  username = "admin"
  # password via the NODEGRID_PASSWORD environment variable
}

resource "nodegrid_settings" "hostname" {
  host = "192.0.2.10"
  settings = {
    "/settings/network_settings/hostname" = "console-server-01"
  }
}
```

## What it can manage

Nodegrid stores its entire configuration as one uniform settings tree, so the
single [`nodegrid_settings`](resources/settings.md) resource is not limited to
any fixed list of features — hostname, DNS, NTP, system preferences, ZPE Cloud
enrollment, network interfaces, serial port attributes, authentication, and
services are all just paths. Run `export_settings /settings/<section>` on a
device to discover the paths for anything not documented here.

For configuration built with `add`/`delete`/`edit` instead of paths — firewall
rules, DHCP scopes, NAT chains, bonding, port renames —
[`nodegrid_exec`](resources/exec.md) runs raw CLI batches.

See the [Common tasks](guides/common-tasks.md) guide for worked examples of
each.

## Schema

### Optional

- `username` (String) SSH/CLI user. Falls back to the `NODEGRID_USERNAME`
  environment variable, then `admin`.
- `password` (String, Sensitive) SSH/CLI password. Falls back to the
  `NODEGRID_PASSWORD` environment variable.
- `port` (Number) SSH port, default `22`. Falls back to `NODEGRID_SSH_PORT`.
- `timeout_seconds` (Number) Per-session SSH timeout, default `30`.
