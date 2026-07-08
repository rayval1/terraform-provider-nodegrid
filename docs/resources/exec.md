---
page_title: "nodegrid_exec Resource - nodegrid"
description: |-
  Runs a batch of raw Nodegrid CLI commands for configuration the settings tree cannot express.
---

# nodegrid_exec (Resource)

Runs a batch of Nodegrid CLI commands over SSH in a single session. Use it
for configuration built with `add`/`delete` rather than settings paths —
firewall rules, DHCP scopes, NAT chains, bonded interfaces, failover groups.

Prefer [`nodegrid_settings`](settings.md) wherever the configuration is a
plain `/settings/...` path: settings get drift detection, exec does not.

The command batch re-runs whenever `commands` changes (the resource is
replaced). Include your own `commit` commands. The provider fails the run if
the CLI reports an error.

## Example Usage

```terraform
resource "nodegrid_exec" "allow_ssh" {
  host = "192.0.2.10"
  commands = [
    "cd /settings/ipv4_firewall/chains/INPUT",
    "add",
    "set rule_number=0 target=ACCEPT protocol=tcp destination_port=22 description=allow-ssh",
    "commit",
  ]
}
```

## Schema

### Required

- `host` (String) Device IP or hostname to SSH into. Changing it forces
  replacement.
- `commands` (List of String) CLI commands executed in order in one session.
  Changing them forces replacement (the new batch runs).

### Optional

- `destroy_commands` (List of String) Commands run on destroy — e.g. delete
  the rules the create batch added. If omitted, destroy leaves the device
  untouched.

### Read-Only

- `output` (String) Session transcript from the last run, for debugging.

## Concurrency

Nodegrid allows only one configuration session per device at a time. Chain
resources that target the same device with `depends_on`, or a concurrent
commit fails with "The system configuration has been changed. Please
revert."
