---
page_title: "nodegrid_settings Data Source - nodegrid"
description: |-
  Reads a Nodegrid settings subtree from a device.
---

# nodegrid_settings (Data Source)

Exports a subtree of a device's live settings via `export_settings` — useful
for inspection or feeding values into other configuration.

## Example Usage

```terraform
data "nodegrid_settings" "net" {
  host = "192.0.2.10"
  path = "/settings/network_settings"
}

output "current_dns" {
  value = data.nodegrid_settings.net.settings["/settings/network_settings/global_dns_servers"]
}
```

## Schema

### Required

- `host` (String) Device IP or hostname to SSH into.
- `path` (String) Settings tree prefix to export, e.g.
  `/settings/network_settings`.

### Read-Only

- `settings` (Map of String) Full setting path to value, as currently
  configured on the device.


### Optional

- `jump_host` (String) Intermediate device to tunnel through, equivalent to
  `ssh -J`. Use for devices on a NAT'd LAN reachable only from a router unit.
  Reached with the same credentials and port as the target; authentication to
  the target is end-to-end, not delegated to the jump host.
