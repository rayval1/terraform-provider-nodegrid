---
page_title: "nodegrid_settings Resource - nodegrid"
description: |-
  Declaratively manages Nodegrid CLI settings on a single device.
---

# nodegrid_settings (Resource)

Manages an arbitrary set of Nodegrid settings-tree values on one device.
Nodegrid configuration is a uniform tree of
`/settings/<section>/<field>=<value>` pairs, so this one resource covers
hostname, DNS, NTP, system preferences, serial port labels, and anything
else expressible as a settings path.

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
