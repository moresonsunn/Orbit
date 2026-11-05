package netflow

import (
	"testing"
)

func TestParseAddrIPv4(t *testing.T) {
	// 0100007F:0050 => 127.0.0.1:80
	ip, port, ok := parseAddr("0100007F:0050", false)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if ip != "127.0.0.1" {
		t.Errorf("expected 127.0.0.1, got %s", ip)
	}
	if port != 80 {
		t.Errorf("expected port 80, got %d", port)
	}

	// 00000000:1F90 => 0.0.0.0:8080 (0x1F90 = 8080)
	ip2, port2, ok2 := parseAddr("00000000:1F90", false)
	if !ok2 {
		t.Fatalf("expected ok2=true")
	}
	if ip2 != "0.0.0.0" {
		t.Errorf("expected 0.0.0.0, got %s", ip2)
	}
	if port2 != 8080 {
		t.Errorf("expected port 8080, got %d", port2)
	}
}

func TestClassifyPort(t *testing.T) {
	if ClassifyPort(80) != "HTTP" {
		t.Errorf("expected HTTP for 80")
	}
	if ClassifyPort(443) != "HTTPS" {
		t.Errorf("expected HTTPS for 443")
	}
	if ClassifyPort(53) != "DNS" {
		t.Errorf("expected DNS for 53")
	}
	if ClassifyPort(8787) != "Orbit" {
		t.Errorf("expected Orbit for 8787")
	}
	if ClassifyPort(9999) != "UNKNOWN" {
		t.Errorf("expected UNKNOWN for 9999")
	}
}

