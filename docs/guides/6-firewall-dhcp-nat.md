---
page_title: "6. Firewall, DHCP and NAT"
subcategory: "Examples"
description: |-
  Configure the INPUT chain, a DHCP scope and NAT masquerading with nodegrid_exec — the parts of the config built with add/delete rather than settings paths.
---

# Firewall, DHCP and NAT

These are list-shaped: rules are created with `add`, not by setting a value at
a known path. `nodegrid_settings` cannot express them, so `nodegrid_exec` runs
the CLI batch directly.

The trade-off is real: **`nodegrid_exec` has no drift detection.** It re-runs
when its `commands` change; it cannot tell you that someone added a firewall
rule by hand. Prefer `nodegrid_settings` wherever the config is a plain path.

## Firewall INPUT chain

```terraform
resource "nodegrid_exec" "firewall" {
  host = "192.0.2.10"
  commands = [
    "cd /settings/ipv4_firewall/chains/INPUT",
    "add",
    "set target=ACCEPT input_interface=lo description=Loopback",
    "add",
    "set target=ACCEPT protocol=icmp description=ICMP",
    "add",
    "set target=ACCEPT protocol=tcp destination_port=22 enable_state_match=yes new=yes description=SSH",
    "add",
    "set target=ACCEPT protocol=tcp destination_port=443 enable_state_match=yes new=yes description=HTTPS",
    "add",
    "set target=ACCEPT enable_state_match=yes related=yes established=yes description=Established",
    "commit",
    "set /settings/ipv4_firewall/policy INPUT=DROP",
    "commit",
  ]
}
```

!> **This batch appends; it does not replace.** Re-running it adds a second
copy of every rule. Chains grow without bound across applies. If you re-push
firewall config regularly, delete the existing rules first or accept the
duplication — a chain with the same seven rules five times over is a common
result of scripting this without a clear step.

~> A `DROP` policy makes closed ports **time out**. A `REJECT` returns
connection-refused. That difference is the fastest way to tell a firewall
block from a service that is down.

## DHCP scope

```terraform
resource "nodegrid_exec" "dhcp" {
  host = "192.0.2.10"
  commands = [
    "cd /settings/dhcp_server",
    "add",
    "set subnet=192.168.0.0 netmask=255.255.255.0 domain=example.com",
    "commit",
    "cd /settings/dhcp_server/192.168.0.0/network_range",
    "add",
    "set ip_address_start=192.168.0.100 ip_address_end=192.168.0.200",
    "commit",
  ]

  depends_on = [nodegrid_exec.firewall]
}
```

## NAT masquerading

Turning a console server into a router for the devices behind it:

```terraform
resource "nodegrid_exec" "nat" {
  host = "192.0.2.10"
  commands = [
    "set /settings/network_settings enable_ipv4_ip_forward=yes",
    "commit",
    "cd /settings/ipv4_nat/chains/POSTROUTING",
    "add",
    "set target=MASQUERADE output_interface=eth0 source_net4=192.168.0.0/24",
    "commit",
  ]

  depends_on = [nodegrid_exec.dhcp]
}
```

Note `enable_ipv4_ip_forward` is a plain settings path — it could equally be
managed by `nodegrid_settings` and get drift detection. Split it out if you
want that.

## Undoing on destroy

`nodegrid_exec` takes optional `destroy_commands`. Without them, destroy
leaves the device untouched:

```terraform
resource "nodegrid_exec" "nat" {
  # ...
  destroy_commands = [
    "cd /settings/ipv4_nat/chains/POSTROUTING",
    "delete 0",
    "commit",
  ]
}
```

## Ordering

Everything above targets one device, and a Nodegrid permits only one
configuration session at a time — a second concurrent session fails with
*"The system configuration has been changed. Please revert."* Chain them with
`depends_on`, as shown. Different devices still apply in parallel.
