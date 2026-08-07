---
page_title: "7. Security hardening: services, access lockout, ZPE Cloud"
subcategory: "Examples"
description: |-
  Disable unused services, set login-failure lockout thresholds, configure the banner, and enrol in ZPE Cloud.
---

# Security hardening

Everything here is a plain settings path, so all of it gets drift detection —
which matters more for hardening than for anything else on these pages. If
someone re-enables telnet by hand, the next `plan` says so.

Field names on this page come from `/settings/services`,
`/settings/system_preferences` and `/settings/zpe_cloud` on real firmware.

## Turn off what you are not using

```terraform
resource "nodegrid_settings" "services" {
  host = "192.0.2.10"
  settings = {
    "/settings/services/enable_telnet_service_to_nodegrid" = "no"
    "/settings/services/enable_ftp_service"                = "no"
    "/settings/services/enable_snmp_service"               = "no"
    "/settings/services/enable_rpc"                        = "no"
    "/settings/services/enable_usb_over_ip"                = "no"

    "/settings/services/ssh_tcp_port"        = "22"
    "/settings/services/ssh_allow_root_access" = "no"
    "/settings/services/enable_http_access"  = "no"
    "/settings/services/enable_https_access" = "yes"
    "/settings/services/https_port"          = "443"
    "/settings/services/redirect_http_to_https" = "no"

    "/settings/services/tlsv1"   = "no"
    "/settings/services/tlsv1.1" = "no"
    "/settings/services/tlsv1.2" = "yes"
    "/settings/services/tlsv1.3" = "yes"
  }
}
```

## Lockout on repeated authentication failures

This one deserves attention because it has bitten people, including while
this provider was being tested:

```terraform
resource "nodegrid_settings" "lockout" {
  host = "192.0.2.10"
  settings = {
    "/settings/services/block_host_with_multiple_authentication_failures"      = "yes"
    "/settings/services/block_host_number_of_authentication_failures_to_block" = "6"
    "/settings/services/block_host_timeframe_to_monitor_authentication_failures" = "1"
    "/settings/services/block_host_period_host_will_stay_blocked"              = "10080"
    "/settings/services/block_host_whitelisted_ip_addresses"                   = "198.51.100.7"

    "/settings/services/block_account_with_multiple_authentication_failures"      = "yes"
    "/settings/services/block_account_number_of_authentication_failures_to_block" = "5"
    "/settings/services/block_account_period_account_will_stay_blocked"           = "30"
  }
}
```

!> **Understand these numbers before applying them.** Periods are in
**minutes** — `10080` is seven days. With a one-minute window and a threshold
of six, six failed logins in a few seconds blocks the source IP for a week,
across **all ports**, not just SSH. A run that opens several parallel sessions
with a bad credential can lock your automation host out of the whole site.

Two defences: whitelist your automation host's egress address in
`block_host_whitelisted_ip_addresses`, and serialise reads with `depends_on`
so a bad credential costs one failure rather than six.

~> A block appears as **connection refused on every port** while ICMP still
answers — it is inserted as a REJECT rule ahead of the chain policy. That
looks nothing like a firewall `DROP`, which times out instead.

## Banner and session timeout

```terraform
resource "nodegrid_settings" "preferences" {
  host = "192.0.2.10"
  settings = {
    "/settings/system_preferences/enable_banner" = "yes"
    "/settings/system_preferences/banner"        = "Authorized access only. All activity is logged."
    "/settings/system_preferences/idle_timeout"  = "36000"
  }
}
```

## ZPE Cloud enrolment

Worth having configured before you need it — it is out-of-band, so it still
works when the device is unreachable over IP:

```terraform
resource "nodegrid_settings" "zpe_cloud" {
  host = "192.0.2.10"
  settings = {
    "/settings/zpe_cloud/enable_zpe_cloud"       = "yes"
    "/settings/zpe_cloud/enable_remote_access"   = "yes"
    "/settings/zpe_cloud/enable_file_protection" = "no"
    "/settings/zpe_cloud/enable_file_encryption" = "no"
  }
}
```

## Related sections

`/settings/password_rules`, `/settings/local_accounts`,
`/settings/authentication`, `/settings/authorization`, `/settings/auditing`
and `/settings/system_logging` are all writable the same way. Export each
before use — the field names in those sections were not verified for this
guide.
