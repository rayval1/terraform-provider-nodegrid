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

resource "nodegrid_settings" "dns" {
  host = "192.0.2.10"
  settings = {
    "/settings/network_settings/global_dns_servers" = "192.0.2.53 198.51.100.53"
    "/settings/network_settings/domain_name"        = "example.com"
  }
}
```

## Schema

### Optional

- `username` (String) SSH/CLI user. Falls back to the `NODEGRID_USERNAME`
  environment variable, then `admin`.
- `password` (String, Sensitive) SSH/CLI password. Falls back to the
  `NODEGRID_PASSWORD` environment variable.
- `port` (Number) SSH port, default `22`. Falls back to `NODEGRID_SSH_PORT`.
- `timeout_seconds` (Number) Per-session SSH timeout, default `30`.
