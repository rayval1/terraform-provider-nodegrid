---
page_title: "1. Device identity: hostname, domain, DNS"
subcategory: "Examples"
description: |-
  Name a device or a whole fleet, set the search domain, and manage global DNS servers — with real drift detection.
---

# Device identity: hostname, domain, DNS

The smallest useful thing the provider does, and the one that shows why it
beats pushing CLI scripts: reads run `export_settings`, so if someone renames
a device in the web UI, the next `terraform plan` shows it and offers to put
it back.

All paths on this page are verified against Nodegrid firmware.

## One device

```terraform
resource "nodegrid_settings" "identity" {
  host = "192.0.2.10"
  settings = {
    "/settings/network_settings/hostname"           = "console-server-01"
    "/settings/network_settings/domain_name"        = "example.com"
    "/settings/network_settings/global_dns_servers" = "192.0.2.53 198.51.100.53"
  }
}
```

`global_dns_servers` is a single space-joined string, not a list. If you keep
servers as a list elsewhere, `join(" ", var.dns_servers)`.

## A fleet, with a naming convention

Derive the name rather than repeating it, so the convention lives in one place:

```terraform
variable "racks" {
  type        = map(string)
  description = "Rack number => management IP."
  default = {
    "101" = "192.0.2.11"
    "102" = "192.0.2.12"
    "105" = "192.0.2.15"
  }
}

variable "router_rack" {
  type    = string
  default = "105"
}

resource "nodegrid_settings" "identity" {
  for_each = var.racks

  host = each.value
  settings = {
    "/settings/network_settings/hostname" = format(
      "console-%s-%s",
      each.key,
      each.key == var.router_rack ? "cc1" : "ce1",
    )
    "/settings/network_settings/domain_name"        = "example.com"
    "/settings/network_settings/global_dns_servers" = "192.0.2.53 198.51.100.53"
  }
}
```

One `for_each` covers the fleet because credentials live on the provider and
the address lives on the resource.

## Confirming what the device thinks

```terraform
data "nodegrid_settings" "check" {
  host = var.racks[var.router_rack]
  path = "/settings/network_settings"

  depends_on = [nodegrid_settings.identity]
}

output "hostname_on_device" {
  value = lookup(
    data.nodegrid_settings.check.settings,
    "/settings/network_settings/hostname",
    "<not reported>",
  )
}
```

## Other fields in this section

`export_settings /settings/network_settings` on real firmware also returns
`enable_ipv4_ip_forward`, `enable_ipv6_ip_forward`, `reverse_path_filtering`,
`enable_multiple_routing_tables`, `enable_vrf_strict_mode`,
`enable_ipv6_segment_routing`, `ipv6_segment_routing_flowlabel`, and
`enable_bluetooth_network` — all writable the same way.
