---
page_title: "nodegrid_settings Resource - nodegrid"
description: |-
  Declaratively manages Nodegrid CLI settings on a single device.
---

# nodegrid_settings (Resource)

Manages an arbitrary set of Nodegrid settings-tree values on one device.
Nodegrid configuration is a uniform tree of
`/settings/<section>/<field>=<value>` pairs, so this one resource covers
hostname, DNS, NTP, system preferences, ZPE Cloud enrollment, network
interfaces, serial port attributes, and anything else expressible as a
settings path. See [Common tasks](../guides/common-tasks.md) for worked
examples.

To find the path for a setting, run `export_settings /settings/<section>` on
the device — the keys it prints are exactly what goes in `settings`.

Note that *renaming* an object (a serial port, for instance) changes its key
in the tree rather than setting a value, so it is not expressible here; use
[`nodegrid_exec`](exec.md) for that, then manage the resulting paths with this
resource.

Reads run `export_settings` against the live device, so configuration drift
appears in `terraform plan`.

## Example Usage

```terraform
resource "nodegrid_settings" "baseline" {
  host = "192.0.2.10"
  settings = {
    "/settings/network_settings/hostname"           = "console-server-01"
    "/settings/network_settings/global_dns_servers" = "192.0.2.53 198.51.100.53"
    "/settings/network_settings/domain_name"        = "example.com"
  }
}
```

## Schema

### Required

- `host` (String) Device IP or hostname to SSH into. Changing it forces
  replacement.
- `settings` (Map of String) Map of full setting path (e.g.
  `/settings/network_settings/domain_name`) to desired value.


### Optional

- `jump_host` (String) Intermediate device to tunnel through, equivalent to
  `ssh -J`. Use for devices on a NAT'd LAN reachable only from a router unit.
  Reached with the same credentials and port as the target; authentication to
  the target is end-to-end, not delegated to the jump host. Changing it forces replacement.

## Behavior notes

- **Destroy does not touch the device.** There is no generic "unset" in the
  Nodegrid settings tree; destroying the resource only removes it from
  Terraform state.
- Removing a key from `settings` likewise stops managing it without
  reverting it on the device.
- Keys that never appear in `export_settings` output (write-only secrets)
  retain their last-known value instead of reporting perpetual drift.
- Only one configuration session may modify a Nodegrid device at a time. If
  you manage the same device from multiple resources, chain them with
  `depends_on` to avoid commit conflicts.
