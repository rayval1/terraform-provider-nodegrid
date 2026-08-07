# NTP and timezone. See docs/guides/2-time-and-ntp.md.
#
# Field names verified via `export_settings /settings/date_and_time`.
# The mode field is `date_and_time`; the server field is `server`.
# There is no `ntp_mode` and no numbered server list.
terraform {
  required_providers {
    nodegrid = { source = "rayval1/nodegrid", version = "~> 0.2" }
  }
}

provider "nodegrid" { username = "admin" }

variable "hosts" {
  type    = map(string)
  default = { "console-01" = "192.0.2.10", "console-02" = "192.0.2.11" }
}

variable "ntp_server" {
  type    = string
  default = "192.0.2.123"
}

resource "nodegrid_settings" "time" {
  for_each = var.hosts

  host = each.value
  settings = {
    "/settings/date_and_time/date_and_time" = "network_time_protocol"
    "/settings/date_and_time/server"        = var.ntp_server
    "/settings/date_and_time/zone"          = "utc"
  }
}
