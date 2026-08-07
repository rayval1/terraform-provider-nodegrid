# Changelog

## v0.2.2

- Replaced site-specific values in tests, code comments, guides and examples
  with RFC 5737 documentation addresses and generic hostnames. Earlier
  versions embedded a real domain, real internal DNS server addresses, and a
  real device hostname captured while validating the parser against live
  hardware. No credentials were ever included.

## v0.2.1

Documentation only — no provider code changes.

- Replaced the single "Common tasks" page with eight registry guides: an
  overview plus seven worked examples (identity/DNS, time/NTP, serial ports,
  network interfaces, jump host, firewall/DHCP/NAT, security hardening). The
  registry renders `docs/`, so this is what a reader actually sees; previously
  the only discoverable example was DNS, which made a general-purpose
  settings-tree provider look DNS-only.
- The overview lists all 58 settings sections a device exposes, to make clear
  the provider is not limited to the documented ones.
- New runnable roots under `examples/`: `ntp`, `network-interfaces`,
  `jump-host`, `firewall-dhcp-nat`, `hardening`.
- Documented the host-lockout fields in `/settings/services`, including that
  periods are in minutes and a block presents as connection-refused on every
  port while ICMP still answers.

## v0.2.0

- **Jump-host support.** `nodegrid_settings`, `nodegrid_exec`, and the
  `nodegrid_settings` data source accept an optional `jump_host`, tunnelling to
  the target the way `ssh -J` does: a TCP connection is opened to the target
  from the jump host, then a second SSH handshake runs over it, so
  authentication to the target is end-to-end rather than delegated. This makes
  devices on a NAT'd LAN behind a router unit manageable — previously the
  provider could only reach directly-routable addresses.

  Verified against a device reachable only through its router unit; the same
  address is unreachable on a direct dial.

## v0.1.3

**Fixes a bug that made every read return nothing.** Validated against live Nodegrid hardware — the first time this provider had
been run against a real device.

- `ParseExport` now understands the format Nodegrid actually emits. Devices
  print `"<section> <field>=<value>"` with a SPACE between the section path and
  the field name; the parser expected `"<section>/<field>=<value>"` and
  discarded every line as CLI chatter. Reads returned an empty map, so data
  sources yielded `{}` and resources reported no drift no matter what changed
  on the device. Keys are normalized to the slash-joined form, so nothing in
  user configuration changes.
- Values quoted by the CLI (e.g. `global_dns_servers="10.0.0.1 10.0.0.2"`) are
  now unquoted on read. Without this, a value written unquoted read back with
  quotes and reported drift on every plan forever.
- Parser tests are now anchored to a verbatim device transcript. The previous
  test asserted an invented format, which is why the bug shipped.
- Corrected NTP settings paths in the guide and complete example: the mode
  field is `date_and_time` and the server field is `server`, not `ntp_mode` /
  `ntp_server_1`. Added guidance to verify any section with `export_settings`
  before writing to it.

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
