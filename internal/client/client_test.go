package client

import (
	"reflect"
	"testing"
)

func TestParseExport(t *testing.T) {
	// Mimics a real CLI session transcript: banner, echoed command, settings
	// lines (some with empty values and embedded '=' in the value), prompt.
	out := "WARNING: unauthorized access prohibited\r\n" +
		"[admin@nodegrid /]# export_settings /settings/network_settings\r\n" +
		"/settings/network_settings/hostname=console-server-01\r\n" +
		"/settings/network_settings/global_dns_servers=192.0.2.53 198.51.100.53\r\n" +
		"/settings/network_settings/domain_name=example.com\r\n" +
		"/settings/network_settings/dns_proxy=\r\n" +
		"/settings/network_settings/banner=key=value stays intact\r\n" +
		"not a setting line\r\n" +
		"[admin@nodegrid /]# exit\r\n"

	got := ParseExport(out)
	want := map[string]string{
		"/settings/network_settings/hostname":           "console-server-01",
		"/settings/network_settings/global_dns_servers": "192.0.2.53 198.51.100.53",
		"/settings/network_settings/domain_name":        "example.com",
		"/settings/network_settings/dns_proxy":          "",
		"/settings/network_settings/banner":             "key=value stays intact",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseExport mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestSplitPath(t *testing.T) {
	section, field, err := SplitPath("/settings/network_settings/hostname")
	if err != nil || section != "/settings/network_settings" || field != "hostname" {
		t.Fatalf("SplitPath: got (%q, %q, %v)", section, field, err)
	}
	for _, bad := range []string{"hostname", "/", "/settings/", "/hostname"} {
		if _, _, err := SplitPath(bad); err == nil {
			t.Errorf("SplitPath(%q): expected error", bad)
		}
	}
}

func TestQuoteValue(t *testing.T) {
	if got := quoteValue(`say "hi" \o/`); got != `"say \"hi\" \\o/"` {
		t.Fatalf("quoteValue: got %s", got)
	}
}

func TestFindCLIError(t *testing.T) {
	if msg := findCLIError("ok\nall good\n"); msg != "" {
		t.Fatalf("expected no error, got %q", msg)
	}
	if msg := findCLIError("something\nError: Invalid value: bogus\n"); msg == "" {
		t.Fatal("expected CLI error to be detected")
	}
}
