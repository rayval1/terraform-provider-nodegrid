# Terraform Provider for ZPE Nodegrid

A community Terraform provider for [ZPE Nodegrid](https://zpesystems.com/) console
servers and services routers. Manage Nodegrid configuration declaratively —
with plan-time diffs, drift detection, and per-key updates — instead of
pushing CLI scripts over SSH by hand.

> Community project, not affiliated with or endorsed by ZPE Systems, Inc.

## How it works

Nodegrid exposes its entire configuration as a tree of
`/settings/<section>/<field>=<value>` pairs. This provider drives the
Nodegrid CLI over SSH — the same interface ZPE's own Ansible collection
automates:

- **Reads** run `export_settings <section>` and parse the pairs, so
  `terraform plan` shows real drift against live device state.
- **Writes** run `cd <section>` / `set field=value` / `commit`.

Because the whole config is one uniform tree, a single generic resource
(`nodegrid_settings`) covers hostname, DNS, NTP, system preferences, ZPE
Cloud enrollment, network connections, serial port labels, and anything else
with a settings path. For configuration built with `add`/`delete` instead of
paths — firewall rules, DHCP scopes, NAT chains, bonding — `nodegrid_exec`
runs raw CLI batches. See [examples/complete](examples/complete/main.tf) for
a full console-server baseline using both.

## Usage

```hcl
terraform {
  required_providers {
    nodegrid = {
      source = "rayval1/nodegrid"
    }
  }
}

provider "nodegrid" {
  username = "admin"          # default; also NODEGRID_USERNAME
  # password via NODEGRID_PASSWORD environment variable (recommended)
}

resource "nodegrid_settings" "dns" {
  host = "192.0.2.10"
  settings = {
    "/settings/network_settings/global_dns_servers" = "192.0.2.53 198.51.100.53"
    "/settings/network_settings/domain_name"        = "example.com"
  }
}
```

Credentials live on the provider; the device address lives on each resource,
so one provider block serves a whole fleet via `for_each`. See
[examples/](examples/) and the registry documentation under [docs/](docs/).

## Semantics & caveats

- **Destroy is a no-op on the device.** There is no generic "unset" in the
  Nodegrid settings tree, and silently reverting console-server config would
  be more dangerous than leaving it in place. Destroying the resource (or
  removing a key from `settings`) stops managing the value, nothing more.
- Keys that `export_settings` never emits (e.g. write-only secrets) keep
  their last-known value instead of reporting false drift.
- SSH host keys are not verified (console servers are frequently reimaged
  and re-keyed).
- Early release: developed against Nodegrid OS 5.x/6.x CLI semantics. Please
  open an issue with a session transcript if `export_settings` output on
  your firmware parses incorrectly.

## Development

```bash
go test ./...
go install .    # then use a dev_overrides block pointing at $GOPATH/bin
```

```hcl
# ~/.terraformrc
provider_installation {
  dev_overrides {
    "rayval1/nodegrid" = "/home/you/go/bin"
  }
  direct {}
}
```

## Roadmap

- `terraform import` support
- Typed convenience resources (`nodegrid_dns`, `nodegrid_serial_ports`)
- First-class list-shaped resources (firewall chains, DHCP scopes) to
  replace `nodegrid_exec` escape hatches

## License

[MPL-2.0](LICENSE). Copyright (c) 2026 Rayval Rodman.
