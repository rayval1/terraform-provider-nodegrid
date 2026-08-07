# Security hardening. See docs/guides/7-security-hardening.md.
#
# All plain settings paths, so all of it gets drift detection — if someone
# re-enables telnet by hand, the next plan says so.
terraform {
  required_providers {
    nodegrid = { source = "rayval1/nodegrid", version = "~> 0.2" }
  }
}

provider "nodegrid" { username = "admin" }

variable "host" {
  type    = string
  default = "192.0.2.10"
}

variable "automation_egress_ip" {
  type        = string
  description = "Whitelisted from host lockout so a bad credential cannot lock automation out of the site."
  default     = "198.51.100.7"
}

resource "nodegrid_settings" "services" {
  host = var.host
  settings = {
    "/settings/services/enable_telnet_service_to_nodegrid" = "no"
    "/settings/services/enable_ftp_service"                = "no"
    "/settings/services/enable_snmp_service"               = "no"
    "/settings/services/enable_rpc"                        = "no"
    "/settings/services/enable_usb_over_ip"                = "no"

    "/settings/services/ssh_tcp_port"          = "22"
    "/settings/services/ssh_allow_root_access" = "no"
    "/settings/services/enable_http_access"    = "no"
    "/settings/services/enable_https_access"   = "yes"
    "/settings/services/https_port"            = "443"

    "/settings/services/tlsv1"   = "no"
    "/settings/services/tlsv1.1" = "no"
    "/settings/services/tlsv1.2" = "yes"
    "/settings/services/tlsv1.3" = "yes"
  }
}

# Periods are in MINUTES. 10080 = 7 days. Read guide 7 before changing these.
resource "nodegrid_settings" "lockout" {
  host = var.host
  settings = {
    "/settings/services/block_host_with_multiple_authentication_failures"        = "yes"
    "/settings/services/block_host_number_of_authentication_failures_to_block"   = "6"
    "/settings/services/block_host_timeframe_to_monitor_authentication_failures" = "1"
    "/settings/services/block_host_period_host_will_stay_blocked"                = "10080"
    "/settings/services/block_host_whitelisted_ip_addresses"                     = var.automation_egress_ip
  }

  depends_on = [nodegrid_settings.services]
}

resource "nodegrid_settings" "zpe_cloud" {
  host = var.host
  settings = {
    "/settings/zpe_cloud/enable_zpe_cloud"     = "yes"
    "/settings/zpe_cloud/enable_remote_access" = "yes"
  }

  depends_on = [nodegrid_settings.lockout]
}
