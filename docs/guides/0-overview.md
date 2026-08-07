---
page_title: "0. Overview: what this provider can manage"
subcategory: "Examples"
description: |-
  The three building blocks, the 58 settings sections they reach, and how to find the field names for anything not documented.
---

# Overview

There are only three building blocks. Nodegrid stores its entire configuration
as one uniform tree, so a single generic resource covers nearly everything —
there is no per-feature resource to wait for.

| | What it does |
|---|---|
| `nodegrid_settings` (resource) | Writes any `/settings/<section>/<field>` value. Reads real state via `export_settings`, so drift appears in `plan`. |
| `nodegrid_exec` (resource) | Runs raw CLI batches for config built with `add`/`delete`/`edit`. No drift detection. |
| `nodegrid_settings` (data source) | Reads a subtree. Never writes — safe against production. |

## The guides

1. [Device identity: hostname, domain, DNS](1-identity-and-dns.md)
2. [Time: NTP and timezone](2-time-and-ntp.md)
3. [Serial ports: naming and access settings](3-serial-ports.md)
4. [Network interfaces: static addressing, DHCP, failover](4-network-interfaces.md)
5. [Devices behind a NAT'd LAN (jump host)](5-devices-behind-nat.md)
6. [Firewall, DHCP and NAT](6-firewall-dhcp-nat.md)
7. [Security hardening: services, access lockout, ZPE Cloud](7-security-hardening.md)

## What that reaches

`ls /settings` on a Nodegrid returns **58 sections**, and `nodegrid_settings`
can write to any field in any of them:

```
auditing            authentication      authorization       auto_discovery
central_management  certificates        cluster             custom_fields
date_and_time       devices             devices_session_preferences
devices_views_preferences               dhcp_relay          dhcp_server
dial_up             fips_140            flow_exporter       frr
geo_fence           hosts               io_ports            ipsec
ipv4_firewall       ipv4_nat            ipv6_firewall       ipv6_nat
license             local_accounts      network_connections network_failover
network_settings    ntp_authentication  ntp_server          password_rules
power_menu          qos                 remote_file_system  routing
scheduler           sdwan               services            sms_settings
sms_whitelist       snmp                ssh_keys            ssl_vpn
static_routes       switch_backplane    switch_global       switch_interfaces
switch_poe          switch_vlan         system_logging      system_preferences
types               wireguard           wireless_modem      zpe_cloud
```

The guides cover perhaps eight of those. Everything else — VLANs, PoE, IPsec,
WireGuard, SNMP, routing, local accounts, audit config — is reachable today
with no change to the provider.

## Finding field names

**Ask the device. Do not guess, and do not trust documentation over the
device** — the tree is firmware-specific:

```
export_settings /settings/<section>
```

The keys it prints are exactly the keys that go in a `settings` map. Guide 2
shows why this matters: the NTP mode field is named `date_and_time`, not
`ntp_mode`, and no reasonable guess would have produced that.

## Limits

- **Renames** (changing an object's key) are not path writes — use `nodegrid_exec`.
- **`add`/`delete` lists** (firewall rules, DHCP scopes) — `nodegrid_exec`, no drift detection.
- **Destroy is a no-op on the device.** There is no generic "unset"; destroying stops managing a value, nothing more.
- **Masked fields** (passwords return as `********`) keep their last-known value instead of reporting false drift.
- **No `terraform import`** yet.
