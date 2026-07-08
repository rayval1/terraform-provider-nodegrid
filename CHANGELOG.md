# Changelog

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
