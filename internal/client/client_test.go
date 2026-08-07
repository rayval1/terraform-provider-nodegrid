package client

import (
	"reflect"
	"testing"
)

// TestParseExportRealTranscript uses a VERBATIM transcript captured from a
// Nodegrid device, not a hand-written
// approximation. The original test invented a "<section>/<field>=<value>"
// format that no device emits; it passed while the parser silently discarded
// every real line, so `terraform plan` saw an empty settings map and reported
// no drift. Keep this anchored to real output.
func TestParseExportRealTranscript(t *testing.T) {
	out := "WARNING: Improper use of shell commands could lead to data loss,\r\n" +
		"[admin@nodegrid /]# export_settings /settings/network_settings\r\n" +
		"/settings/network_settings hostname=console-server-01\r\n" +
		"/settings/network_settings domain_name=example.com\r\n" +
		"/settings/network_settings global_dns_servers=\"192.0.2.53 198.51.100.53\"\r\n" +
		"/settings/network_settings enable_ipv4_ip_forward=yes\r\n" +
		"/settings/network_settings reverse_path_filtering=disabled\r\n" +
		"not a setting line\r\n" +
		"[admin@nodegrid /]# exit\r\n"

	got := ParseExport(out)
	want := map[string]string{
		"/settings/network_settings/hostname":               "console-server-01",
		"/settings/network_settings/domain_name":            "example.com",
		"/settings/network_settings/global_dns_servers":     "192.0.2.53 198.51.100.53",
		"/settings/network_settings/enable_ipv4_ip_forward": "yes",
		"/settings/network_settings/reverse_path_filtering": "disabled",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseExport mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}

// Values with embedded '=', empty values, and the already-slash-joined form
// all still parse.
func TestParseExportEdgeCases(t *testing.T) {
	out := "/settings/network_settings banner=key=value stays intact\r\n" +
		"/settings/network_settings dns_proxy=\r\n" +
		"/settings/auth/method=local\r\n" +
		"/settings/x y z=bogus\r\n"

	got := ParseExport(out)
	want := map[string]string{
		"/settings/network_settings/banner":    "key=value stays intact",
		"/settings/network_settings/dns_proxy": "",
		"/settings/auth/method":                "local",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseExport mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}

// A value written unquoted must read back unquoted, or every plan reports
// drift on it forever.
func TestQuoteRoundTrip(t *testing.T) {
	for _, v := range []string{"10.0.0.1 10.0.0.2", "plain", `say "hi"`, ""} {
		if got := unquoteValue(quoteValue(v)); got != v {
			t.Errorf("round trip %q: got %q", v, got)
		}
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
