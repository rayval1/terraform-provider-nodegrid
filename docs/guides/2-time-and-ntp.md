---
page_title: "2. Time: NTP and timezone"
subcategory: "Examples"
description: |-
  Point a device at an NTP server and set its timezone — and why the field names here are not what you would guess.
---

# Time: NTP and timezone

Paths on this page are verified against Nodegrid firmware. **Read the warning
before copying anything** — this section is the best example of why guessing
field names fails.

```terraform
resource "nodegrid_settings" "time" {
  host = "192.0.2.10"
  settings = {
    "/settings/date_and_time/date_and_time" = "network_time_protocol"
    "/settings/date_and_time/server"        = "192.0.2.123"
    "/settings/date_and_time/zone"          = "utc"
  }
}
```

~> **The field names are counterintuitive.** The mode field is literally
`date_and_time` — the section and the field share a name — and the server
field is just `server`. There is **no** `ntp_mode`, no `ntp_server_1`, and no
numbered list of servers. Configuration written against those guessed names
fails on the device.

The whole subtree is three fields:

```
/settings/date_and_time date_and_time=network_time_protocol
/settings/date_and_time server=10.0.0.23
/settings/date_and_time zone=utc
```

## The general lesson

Before writing to any section, ask the device what it actually has:

```
export_settings /settings/date_and_time
```

The keys it prints are exactly the keys that go in a `settings` map. This is
the authoritative source — more reliable than documentation, including this
page, because the tree is firmware-specific.

## Related sections

Authenticated NTP and running the device *as* an NTP server live in separate
sections, `/settings/ntp_authentication` and `/settings/ntp_server`. Export
those before use; their fields are not documented here.

## Fleet-wide

```terraform
resource "nodegrid_settings" "time" {
  for_each = var.racks

  host = each.value
  settings = {
    "/settings/date_and_time/date_and_time" = "network_time_protocol"
    "/settings/date_and_time/server"        = var.ntp_server
    "/settings/date_and_time/zone"          = "utc"
  }
}
```
