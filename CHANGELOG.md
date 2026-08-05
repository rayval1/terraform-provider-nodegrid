# Changelog

## v0.1.2

Documentation and examples only — no provider code changes.

- New "Common tasks" guide covering hostname, DNS, NTP, system preferences,
  ZPE Cloud, static addressing, serial ports, firewall rules, and same-device
  ordering. Published to the registry docs, which render `docs/` rather than
  `examples/`.
- New `examples/hostname` (single device and fleet-wide, with drift check)
  and `examples/serial-ports`.
- Clarified in `nodegrid_settings` docs that *renaming* a tree object (e.g. a
  serial port) is not a settings-path write and needs `nodegrid_exec`; the
  previous wording implied `nodegrid_settings` handled port labels outright.
- Provider index now states that the settings tree is open-ended rather than
  limited to the documented sections, and explains discovering paths with
  `export_settings`.

## v0.1.1

- New `nodegrid_exec` resource: run raw CLI command batches for
  configuration the settings tree cannot express (firewall rules, DHCP
  scopes, NAT chains, bonding), with optional `destroy_commands`.
- New complete example covering a full console-server baseline: hostname,
  DNS, NTP, system preferences, ZPE Cloud, static management interface,
  serial port labels, firewall + DHCP.

## v0.1.0

Initial release.

- `nodegrid_settings` resource: declaratively manage any
  `/settings/<section>/<field>` value on a Nodegrid device over SSH, with
  drift detection via `export_settings`.
- `nodegrid_settings` data source: read a live settings subtree.
- Provider configuration: `username`, `password`, `port`, `timeout_seconds`
  with `NODEGRID_USERNAME` / `NODEGRID_PASSWORD` / `NODEGRID_SSH_PORT`
  environment fallbacks.
