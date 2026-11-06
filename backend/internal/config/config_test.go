package config

import (
	"os"
	"testing"
)

func TestParseListenAddr(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedHost string
		expectedPort string
		expectErr    bool
	}{
		// Bare ports
		{name: "bare port standard", input: "8787", expectedHost: "", expectedPort: "8787", expectErr: false},
		{name: "bare port custom", input: "8080", expectedHost: "", expectedPort: "8080", expectErr: false},
		{name: "bare port with whitespace and slash", input: "  8787/  ", expectedHost: "", expectedPort: "8787", expectErr: false},

		// Colon ports
		{name: "leading colon standard", input: ":8787", expectedHost: "", expectedPort: "8787", expectErr: false},
		{name: "leading colon custom", input: ":9000", expectedHost: "", expectedPort: "9000", expectErr: false},
		{name: "leading colon with whitespace and trailing slash", input: "  :8787/ \r\n", expectedHost: "", expectedPort: "8787", expectErr: false},

		// Host:Port combinations
		{name: "all interfaces 0.0.0.0", input: "0.0.0.0:8787", expectedHost: "0.0.0.0", expectedPort: "8787", expectErr: false},
		{name: "loopback 127.0.0.1", input: "127.0.0.1:8787", expectedHost: "127.0.0.1", expectedPort: "8787", expectErr: false},
		{name: "custom ip and port", input: "192.168.1.100:9999", expectedHost: "192.168.1.100", expectedPort: "9999", expectErr: false},
		{name: "bracketed ipv6", input: "[::]:8787", expectedHost: "[::]", expectedPort: "8787", expectErr: false},
		{name: "bracketed ipv6 loopback", input: "[::1]:8080", expectedHost: "[::1]", expectedPort: "8080", expectErr: false},
		{name: "unbracketed ipv6 loopback with port", input: "::1:8787", expectedHost: "[::1]", expectedPort: "8787", expectErr: false},

		// Host only (no port)
		{name: "host only 0.0.0.0", input: "0.0.0.0", expectedHost: "0.0.0.0", expectedPort: "", expectErr: false},
		{name: "host only localhost", input: "localhost", expectedHost: "localhost", expectedPort: "", expectErr: false},

		// Empty input
		{name: "empty string", input: "", expectedHost: "", expectedPort: "", expectErr: false},
		{name: "whitespace only", input: "   \t  ", expectedHost: "", expectedPort: "", expectErr: false},

		// Port boundaries
		{name: "min valid port 1", input: ":1", expectedHost: "", expectedPort: "1", expectErr: false},
		{name: "max valid port 65535", input: ":65535", expectedHost: "", expectedPort: "65535", expectErr: false},

		// Invalid port numbers
		{name: "zero port", input: ":0", expectErr: true},
		{name: "bare zero port", input: "0", expectErr: true},
		{name: "port above max 65536", input: ":65536", expectErr: true},
		{name: "port 70000", input: "70000", expectErr: true},
		{name: "negative port", input: ":-80", expectErr: true},
		{name: "invalid characters", input: "invalid", expectErr: true},
		{name: "invalid colon port", input: ":abc", expectErr: true},

		// Malformed inputs and URLs rejected
		{name: "url scheme http", input: "http://localhost:8787", expectErr: true},
		{name: "url scheme https", input: "https://0.0.0.0:8787", expectErr: true},
		{name: "url scheme bare host", input: "http://localhost", expectErr: true},
		{name: "duplicate port colons", input: "0.0.0.0:8787:8787", expectErr: true},
		{name: "double colon before port", input: "127.0.0.1::8787", expectErr: true},
		{name: "bracketed non-ipv6 with port", input: "[0.0.0.0:8787]:8787", expectErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host, port, err := ParseListenAddr(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil (host=%q, port=%q)", tc.input, host, port)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.input, err)
			}
			if host != tc.expectedHost {
				t.Errorf("host mismatch for %q: expected %q, got %q", tc.input, tc.expectedHost, host)
			}
			if port != tc.expectedPort {
				t.Errorf("port mismatch for %q: expected %q, got %q", tc.input, tc.expectedPort, port)
			}
		})
	}
}

func TestResolveListenConfig_Precedence(t *testing.T) {
	tests := []struct {
		name               string
		orbitListen        string
		orbitPort          string
		port               string
		expectedListenAddr string
		expectedPort       string
		expectErr          bool
	}{
		// Default fallback
		{
			name:               "all empty defaults to :8787",
			orbitListen:        "",
			orbitPort:          "",
			port:               "",
			expectedListenAddr: ":8787",
			expectedPort:       "8787",
		},

		// Bare port normalization in ORBIT_LISTEN
		{
			name:               "bare port in ORBIT_LISTEN normalizes to :8787",
			orbitListen:        "8787",
			orbitPort:          "",
			port:               "",
			expectedListenAddr: ":8787",
			expectedPort:       "8787",
		},
		{
			name:               "custom bare port in ORBIT_LISTEN normalizes to :9090",
			orbitListen:        "9090",
			orbitPort:          "",
			port:               "",
			expectedListenAddr: ":9090",
			expectedPort:       "9090",
		},

		// Leading colon in ORBIT_LISTEN
		{
			name:               "colon port :8787 in ORBIT_LISTEN",
			orbitListen:        ":8787",
			orbitPort:          "",
			port:               "",
			expectedListenAddr: ":8787",
			expectedPort:       "8787",
		},

		// Host:Port in ORBIT_LISTEN
		{
			name:               "host:port in ORBIT_LISTEN preserved cleanly",
			orbitListen:        "0.0.0.0:8787",
			orbitPort:          "",
			port:               "",
			expectedListenAddr: "0.0.0.0:8787",
			expectedPort:       "8787",
		},
		{
			name:               "loopback host:port in ORBIT_LISTEN",
			orbitListen:        "127.0.0.1:8787",
			orbitPort:          "",
			port:               "",
			expectedListenAddr: "127.0.0.1:8787",
			expectedPort:       "8787",
		},
		{
			name:               "unbracketed ipv6 loopback in ORBIT_LISTEN resolves with brackets",
			orbitListen:        "::1:8787",
			orbitPort:          "",
			port:               "",
			expectedListenAddr: "[::1]:8787",
			expectedPort:       "8787",
		},

		// PORT override when ORBIT_LISTEN is empty or default
		{
			name:               "PORT overrides default when ORBIT_LISTEN is empty",
			orbitListen:        "",
			orbitPort:          "",
			port:               "8080",
			expectedListenAddr: ":8080",
			expectedPort:       "8080",
		},
		{
			name:               "PORT with colon overrides default",
			orbitListen:        "",
			orbitPort:          "",
			port:               ":8080",
			expectedListenAddr: ":8080",
			expectedPort:       "8080",
		},
		{
			name:               "PORT overrides default Dockerfile ENV ORBIT_LISTEN=:8787",
			orbitListen:        ":8787",
			orbitPort:          "",
			port:               "8080",
			expectedListenAddr: ":8080",
			expectedPort:       "8080",
		},
		{
			name:               "PORT overrides default bare ORBIT_LISTEN=8787",
			orbitListen:        "8787",
			orbitPort:          "",
			port:               "8080",
			expectedListenAddr: ":8080",
			expectedPort:       "8080",
		},

		// ORBIT_PORT override
		{
			name:               "ORBIT_PORT overrides default when ORBIT_LISTEN is empty",
			orbitListen:        "",
			orbitPort:          "9000",
			port:               "",
			expectedListenAddr: ":9000",
			expectedPort:       "9000",
		},
		{
			name:               "ORBIT_PORT overrides default ORBIT_LISTEN=:8787",
			orbitListen:        ":8787",
			orbitPort:          "9000",
			port:               "",
			expectedListenAddr: ":9000",
			expectedPort:       "9000",
		},

		// Precedence: ORBIT_PORT > PORT
		{
			name:               "ORBIT_PORT takes precedence over PORT",
			orbitListen:        "",
			orbitPort:          "9000",
			port:               "8080",
			expectedListenAddr: ":9000",
			expectedPort:       "9000",
		},
		{
			name:               "ORBIT_PORT takes precedence over PORT with default ORBIT_LISTEN",
			orbitListen:        ":8787",
			orbitPort:          "9000",
			port:               "8080",
			expectedListenAddr: ":9000",
			expectedPort:       "9000",
		},

		// Explicit custom ORBIT_LISTEN precedence
		{
			name:               "Explicit custom ORBIT_LISTEN takes precedence over ORBIT_PORT and PORT",
			orbitListen:        ":9999",
			orbitPort:          "9000",
			port:               "8080",
			expectedListenAddr: ":9999",
			expectedPort:       "9999",
		},
		{
			name:               "Explicit custom host:port ORBIT_LISTEN takes precedence",
			orbitListen:        "0.0.0.0:9999",
			orbitPort:          "9000",
			port:               "8080",
			expectedListenAddr: "0.0.0.0:9999",
			expectedPort:       "9999",
		},

		// Host-only ORBIT_LISTEN with port overrides
		{
			name:               "Host-only ORBIT_LISTEN with PORT override",
			orbitListen:        "0.0.0.0",
			orbitPort:          "",
			port:               "9000",
			expectedListenAddr: "0.0.0.0:9000",
			expectedPort:       "9000",
		},
		{
			name:               "Host-only ORBIT_LISTEN without port override defaults to 8787",
			orbitListen:        "127.0.0.1",
			orbitPort:          "",
			port:               "",
			expectedListenAddr: "127.0.0.1:8787",
			expectedPort:       "8787",
		},

		// Error conditions
		{
			name:        "invalid ORBIT_PORT returns error",
			orbitListen: "",
			orbitPort:   "invalid",
			port:        "",
			expectErr:   true,
		},
		{
			name:        "invalid PORT returns error",
			orbitListen: "",
			orbitPort:   "",
			port:        "70000",
			expectErr:   true,
		},
		{
			name:        "invalid ORBIT_LISTEN returns error",
			orbitListen: "invalid:address:format",
			orbitPort:   "",
			port:        "",
			expectErr:   true,
		},
		{
			name:        "URL in ORBIT_LISTEN returns error",
			orbitListen: "http://localhost:8787",
			orbitPort:   "",
			port:        "",
			expectErr:   true,
		},
		{
			name:        "duplicate port in ORBIT_LISTEN returns error",
			orbitListen: "0.0.0.0:8787:8787",
			orbitPort:   "",
			port:        "",
			expectErr:   true,
		},
		{
			name:        "double colon in ORBIT_LISTEN returns error",
			orbitListen: "127.0.0.1::8787",
			orbitPort:   "",
			port:        "",
			expectErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			addr, port, err := ResolveListenConfig(tc.orbitListen, tc.orbitPort, tc.port)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil (addr=%q, port=%q)", addr, port)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if addr != tc.expectedListenAddr {
				t.Errorf("address mismatch: expected %q, got %q", tc.expectedListenAddr, addr)
			}
			if port != tc.expectedPort {
				t.Errorf("port mismatch: expected %q, got %q", tc.expectedPort, port)
			}
		})
	}
}

func TestNormalizeListenAddr_SafeFallback(t *testing.T) {
	// Ensures NormalizeListenAddr never crashes net.Listen even on invalid input
	tests := []struct {
		inputListen string
		inputOP     string
		inputP      string
		expected    string
	}{
		{"8787", "", "", ":8787"},
		{":8787", "", "", ":8787"},
		{"0.0.0.0:8787", "", "", "0.0.0.0:8787"},
		{"[::]:8787", "", "", "[::]:8787"},
		{"::1:8787", "", "", "[::1]:8787"},
		{"invalid", "", "", ":8787"},   // fallback to default
		{":70000", "", "", ":8787"},    // fallback to default
		{"http://localhost:8787", "", "", ":8787"},
		{"0.0.0.0:8787:8787", "", "", ":8787"},
		{"127.0.0.1::8787", "", "", ":8787"},
		{"[0.0.0.0:8787]:8787", "", "", ":8787"},
		{"", "9000", "", ":9000"},
		{"", "", "8080", ":8080"},
	}

	for _, tc := range tests {
		result := NormalizeListenAddr(tc.inputListen, tc.inputOP, tc.inputP)
		if result != tc.expected {
			t.Errorf("NormalizeListenAddr(%q, %q, %q): expected %q, got %q",
				tc.inputListen, tc.inputOP, tc.inputP, tc.expected, result)
		}
	}
}

func TestResolvePort(t *testing.T) {
	tests := []struct {
		inputListen string
		inputOP     string
		inputP      string
		expected    string
	}{
		{"8787", "", "", "8787"},
		{":8787", "", "", "8787"},
		{"0.0.0.0:8787", "", "", "8787"},
		{"http://localhost:8787", "", "", "8787"},
		{"0.0.0.0:8787:8787", "", "", "8787"},
		{"127.0.0.1::8787", "", "", "8787"},
		{"", "9000", "", "9000"},
		{"", "", "8080", "8080"},
		{"invalid", "", "", "8787"},
	}

	for _, tc := range tests {
		result := ResolvePort(tc.inputListen, tc.inputOP, tc.inputP)
		if result != tc.expected {
			t.Errorf("ResolvePort(%q, %q, %q): expected %q, got %q",
				tc.inputListen, tc.inputOP, tc.inputP, tc.expected, result)
		}
	}
}

func TestHealthcheckResolution(t *testing.T) {
	// Save existing env
	oldListen := os.Getenv("ORBIT_LISTEN")
	oldOrbitPort := os.Getenv("ORBIT_PORT")
	oldPort := os.Getenv("PORT")
	defer func() {
		os.Setenv("ORBIT_LISTEN", oldListen)
		os.Setenv("ORBIT_PORT", oldOrbitPort)
		os.Setenv("PORT", oldPort)
	}()

	// Case 1: All unset -> defaults
	os.Unsetenv("ORBIT_LISTEN")
	os.Unsetenv("ORBIT_PORT")
	os.Unsetenv("PORT")

	port := ResolveHealthcheckPort()
	if port != "8787" {
		t.Errorf("expected default healthcheck port '8787', got %q", port)
	}
	url := ResolveHealthcheckURL()
	expectedURL := "http://127.0.0.1:8787/api/system/health"
	if url != expectedURL {
		t.Errorf("expected healthcheck URL %q, got %q", expectedURL, url)
	}

	// Case 2: Custom PORT set
	os.Setenv("PORT", "9090")
	port = ResolveHealthcheckPort()
	if port != "9090" {
		t.Errorf("expected healthcheck port '9090', got %q", port)
	}
	url = ResolveHealthcheckURL()
	if url != "http://127.0.0.1:9090/api/system/health" {
		t.Errorf("expected healthcheck URL targeting 9090, got %q", url)
	}

	// Case 3: ORBIT_PORT overrides PORT
	os.Setenv("ORBIT_PORT", "9191")
	port = ResolveHealthcheckPort()
	if port != "9191" {
		t.Errorf("expected healthcheck port '9191', got %q", port)
	}
}

func TestLoad_Integration(t *testing.T) {
	oldListen := os.Getenv("ORBIT_LISTEN")
	oldOrbitPort := os.Getenv("ORBIT_PORT")
	oldPort := os.Getenv("PORT")
	defer func() {
		os.Setenv("ORBIT_LISTEN", oldListen)
		os.Setenv("ORBIT_PORT", oldOrbitPort)
		os.Setenv("PORT", oldPort)
	}()

	// 1. Bare port 8787 in ORBIT_LISTEN
	os.Setenv("ORBIT_LISTEN", "8787")
	os.Unsetenv("ORBIT_PORT")
	os.Unsetenv("PORT")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Listen != ":8787" {
		t.Errorf("expected cfg.Listen to be ':8787', got %q", cfg.Listen)
	}

	// 2. Custom host:port
	os.Setenv("ORBIT_LISTEN", "0.0.0.0:8787")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Listen != "0.0.0.0:8787" {
		t.Errorf("expected cfg.Listen to be '0.0.0.0:8787', got %q", cfg.Listen)
	}

	// 3. Invalid port should fail Load()
	os.Setenv("PORT", "invalid")
	os.Unsetenv("ORBIT_LISTEN")
	_, err = Load()
	if err == nil {
		t.Fatal("expected Load() to fail on invalid PORT, got nil")
	}
}
