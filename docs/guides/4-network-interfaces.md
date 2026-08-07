---
page_title: "4. Network interfaces: static addressing, DHCP, failover"
subcategory: "Examples"
description: |-
  Give an interface a static management address, set route metrics, and re-address safely without locking yourself out.
---

# Network interfaces

Every interface is a subtree under `/settings/network_connections/<NAME>` —
`ETH0`, `BOND0`, `BACKPLANE0`, `SFP0`, and so on. Field names on this page
are verified against Nodegrid firmware.

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
    "/settings/network_connections/ETH0/ipv4_dns_server"       = "192.0.2.53"
  }
}
```

`ipv4_bitmask` is a prefix length as a string (`"24"`), not a dotted mask.

## DHCP with a pinned route metric

Useful when a device has several uplinks and you care which one wins the
default route — lower metric wins:

```terraform
resource "nodegrid_settings" "bond0" {
  host = "192.0.2.10"
  settings = {
    "/settings/network_connections/BOND0/ipv4_mode"                  = "dhcp"
    "/settings/network_connections/BOND0/ipv4_default_route_metric"  = "90"
    "/settings/network_connections/BOND0/ipv4_ignore_obtained_dns_server" = "yes"
    "/settings/network_connections/BOND0/connect_automatically"      = "yes"
  }
}
```

Also available: `ipv4_ignore_obtained_default_gateway`, `ipv4_dns_search`,
`ipv6_mode` and the matching `ipv6_*` fields.

## Re-addressing without locking yourself out

~> **An address change commits immediately and can drop the SSH session
Terraform is connected over.** Two rules:

1. Re-address the interface you are **not** connected through where possible.
2. Never split moving one address between two interfaces across two
   resources. An IPv4 address cannot exist on two interfaces at once, so if
   one resource clears it from `ETH0` and another sets it on `BOND0`, the
   device ends up holding it on neither and the run dies with the box
   unreachable.

For a move, put both interfaces in a single `nodegrid_exec` so they land on
one `commit`:

```terraform
resource "nodegrid_exec" "migrate_address" {
  host = "192.0.2.10"
  commands = [
    "cd /settings/network_connections/ETH0",
    "set ipv4_mode=static ipv4_address=198.51.100.10 ipv4_bitmask=26 ipv4_gateway=198.51.100.1",
    "cd /settings/network_connections/BOND0",
    "set ipv4_mode=static ipv4_address=192.0.2.10 ipv4_bitmask=24",
    "commit",
  ]
}
```

Expect the session to drop the moment that commits — by design. The device
will be reachable at its new address.

## Failover

Ping-based failover between uplinks lives in
`/settings/network_failover/connections`. Export it before writing:

```
export_settings /settings/network_failover
```
