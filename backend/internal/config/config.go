// Package config loads Orbit's static configuration from environment variables.
//
// Runtime-adjustable settings (intervals, retention, alert thresholds) live in the
// database (see package settings); this package only covers values that must be
// known at process start or that are security relevant and therefore must not be
// changeable from the web UI (e.g. enabling management actions).
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultPort is the standardized project-wide default HTTP port for Orbit.
	DefaultPort = "8787"
	// DefaultListenAddr is the standardized project-wide default listen address.
	DefaultListenAddr = ":" + DefaultPort
)

type Config struct {
	Listen  string
	DataDir string

	// DockerHost is passed to the Docker SDK (unix:///var/run/docker.sock by default,
	// tcp://socket-proxy:2375 when a socket proxy is used).
	DockerHost string

	// Host filesystem mount points (bind mounted read-only into the container).
	HostProc   string
	HostSys    string
	HostEtc    string
	HostRootFS string

	// Feature gates. These are intentionally env-only.
	EnableActions     bool
	EnableDestructive bool
	EBPFMode          string // auto | off
	GPUMode           string // auto | off

	// Security.
	TrustedOrigins []string
	TrustProxy     bool
	SecureCookies  string // auto | true | false
	SessionTTL     time.Duration
	SessionIdle    time.Duration

	// Privilege handling (see internal/privdrop).
	RunAsUID int
	RunAsGID int
	KeepRoot bool

	MaxStatsStreams int
	LogLevel        string
	DevMode         bool
}

func Load() (*Config, error) {
	listenAddr, _, err := ResolveListenConfig(
		os.Getenv("ORBIT_LISTEN"),
		os.Getenv("ORBIT_PORT"),
		os.Getenv("PORT"),
	)
	if err != nil {
		return nil, fmt.Errorf("configuration error: %w", err)
	}

	c := &Config{
		Listen:            listenAddr,
		DataDir:           env("ORBIT_DATA_DIR", "/data"),
		DockerHost:        env("ORBIT_DOCKER_HOST", env("DOCKER_HOST", "unix:///var/run/docker.sock")),
		HostProc:          env("ORBIT_HOST_PROC", "/host/proc"),
		HostSys:           env("ORBIT_HOST_SYS", "/host/sys"),
		HostEtc:           env("ORBIT_HOST_ETC", "/host/etc"),
		HostRootFS:        env("ORBIT_HOST_ROOTFS", "/host/rootfs"),
		EnableActions:     envBool("ORBIT_ENABLE_ACTIONS", false),
		EnableDestructive: envBool("ORBIT_ENABLE_DESTRUCTIVE", false),
		EBPFMode:          strings.ToLower(env("ORBIT_EBPF", "auto")),
		GPUMode:           strings.ToLower(env("ORBIT_GPU", "auto")),
		TrustProxy:        envBool("ORBIT_TRUST_PROXY", false),
		SecureCookies:     strings.ToLower(env("ORBIT_SECURE_COOKIES", "auto")),
		RunAsUID:          envInt("ORBIT_UID", 10001),
		RunAsGID:          envInt("ORBIT_GID", 10001),
		KeepRoot:          envBool("ORBIT_KEEP_ROOT", false),
		MaxStatsStreams:   envInt("ORBIT_MAX_STATS_STREAMS", 250),
		LogLevel:          strings.ToLower(env("ORBIT_LOG_LEVEL", "info")),
		DevMode:           envBool("ORBIT_DEV", false),
	}
	for _, o := range strings.Split(env("ORBIT_TRUSTED_ORIGINS", ""), ",") {
		if o = strings.TrimSpace(strings.TrimRight(o, "/")); o != "" {
			c.TrustedOrigins = append(c.TrustedOrigins, strings.ToLower(o))
		}
	}
	if c.SessionTTL, err = envDuration("ORBIT_SESSION_TTL", 7*24*time.Hour); err != nil {
		return nil, err
	}
	if c.SessionIdle, err = envDuration("ORBIT_SESSION_IDLE", 12*time.Hour); err != nil {
		return nil, err
	}
	if c.EnableDestructive && !c.EnableActions {
		return nil, fmt.Errorf("ORBIT_ENABLE_DESTRUCTIVE requires ORBIT_ENABLE_ACTIONS=true")
	}
	switch c.EBPFMode {
	case "auto", "off", "on":
	default:
		return nil, fmt.Errorf("ORBIT_EBPF must be auto, on or off")
	}
	switch c.GPUMode {
	case "auto", "off":
	default:
		return nil, fmt.Errorf("ORBIT_GPU must be auto or off")
	}
	return c, nil
}

func env(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

func envBool(k string, def bool) bool {
	v, ok := os.LookupEnv(k)
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return b
}

func envInt(k string, def int) int {
	v, ok := os.LookupEnv(k)
	if !ok {
		return def
	}
	i, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return i
}

func envDuration(k string, def time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(k)
	if !ok || strings.TrimSpace(v) == "" {
		return def, nil
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", k, err)
	}
	return d, nil
}

// ParseListenAddr parses and normalizes a listen address string into host and port components.
// It handles bare ports ("8787"), leading colons (":8787"), host:port pairs ("0.0.0.0:8787",
// "127.0.0.1:8787"), bracketed IPv6 ("[::]:8787"), host-only strings, and cleans whitespace
// or trailing slashes.
func ParseListenAddr(s string) (host string, port string, err error) {
	clean := strings.Trim(s, " \t\r\n/")
	if clean == "" {
		return "", "", nil
	}

	// Reject URL schemes (e.g. "http://localhost:8787")
	if strings.Contains(clean, "://") {
		return "", "", fmt.Errorf("invalid listen address format %q: URLs with scheme (e.g. http://) are not allowed", s)
	}

	// 1. Bare numeric port (e.g. "8787")
	if num, err := strconv.Atoi(clean); err == nil {
		if num < 1 || num > 65535 {
			return "", "", fmt.Errorf("port %d out of range [1-65535]", num)
		}
		return "", strconv.Itoa(num), nil
	}

	// 2. Colon port (e.g. ":8787")
	if strings.HasPrefix(clean, ":") && !strings.HasPrefix(clean, "::") {
		candidate := strings.TrimPrefix(clean, ":")
		num, err := strconv.Atoi(candidate)
		if err != nil || num < 1 || num > 65535 {
			return "", "", fmt.Errorf("invalid port in listen address %q", clean)
		}
		return "", strconv.Itoa(num), nil
	}

	// 3. Host:Port (e.g. "0.0.0.0:8787", "127.0.0.1:8787", "[::]:8787")
	h, p, splitErr := net.SplitHostPort(clean)
	if splitErr == nil {
		num, err := strconv.Atoi(p)
		if err != nil || num < 1 || num > 65535 {
			return "", "", fmt.Errorf("invalid port in listen address %q: %v", clean, err)
		}
		if strings.Contains(h, ":") {
			ip := net.ParseIP(h)
			if ip == nil || ip.To4() != nil {
				return "", "", fmt.Errorf("invalid IPv6 address in listen address %q", clean)
			}
			if !strings.HasPrefix(h, "[") {
				h = "[" + h + "]"
			}
		}
		return h, strconv.Itoa(num), nil
	}

	// 4. Bracketed IPv6 fallback (e.g. "[::]:8787" if SplitHostPort had issues)
	if lastColon := strings.LastIndex(clean, ":"); lastColon != -1 {
		candidatePort := strings.Trim(clean[lastColon+1:], " \t\r\n/]")
		candidateHost := strings.Trim(clean[:lastColon], " \t\r\n/[]")
		if num, err := strconv.Atoi(candidatePort); err == nil && num >= 1 && num <= 65535 {
			ip := net.ParseIP(candidateHost)
			if ip != nil && ip.To4() == nil {
				return "[" + candidateHost + "]", strconv.Itoa(num), nil
			}
		}
	}

	// 5. Hostname or IP without port (e.g. "0.0.0.0", "127.0.0.1", "localhost")
	if !strings.Contains(clean, ":") {
		if net.ParseIP(clean) != nil || clean == "localhost" {
			return clean, "", nil
		}
	}

	return "", "", fmt.Errorf("invalid listen address format: %q", s)
}

func parsePortString(s string) (string, error) {
	clean := strings.Trim(s, " \t\r\n/")
	if clean == "" {
		return "", nil
	}
	clean = strings.TrimPrefix(clean, ":")
	num, err := strconv.Atoi(clean)
	if err != nil || num < 1 || num > 65535 {
		return "", fmt.Errorf("invalid port value %q: must be integer between 1 and 65535", s)
	}
	return strconv.Itoa(num), nil
}

// ResolveListenConfig resolves the normalized listen address and port number.
// Precedence:
//   1. ORBIT_LISTEN if explicitly specified with a custom address/port (non-default and non-empty)
//   2. ORBIT_PORT if set (normalized)
//   3. PORT if set (normalized)
//   4. Default ":8787"
// Ensures net.Listen("tcp", addr) will not fail with "missing port in address".
func ResolveListenConfig(orbitListen, orbitPort, port string) (listenAddr string, portNum string, err error) {
	var host string
	var resolvedPort string

	cleanListen := strings.Trim(orbitListen, " \t\r\n/")
	if cleanListen != "" {
		h, p, parseErr := ParseListenAddr(cleanListen)
		if parseErr != nil {
			return "", "", parseErr
		}
		host = h
		if p != "" && p != DefaultPort {
			resolvedPort = p
		} else if p == DefaultPort {
			resolvedPort = DefaultPort
		}
	}

	// ORBIT_PORT override (highest priority for port when ORBIT_LISTEN is default or only sets host)
	cleanOP := strings.Trim(orbitPort, " \t\r\n/")
	if cleanOP != "" {
		op, err := parsePortString(cleanOP)
		if err != nil {
			return "", "", fmt.Errorf("ORBIT_PORT: %w", err)
		}
		if resolvedPort == "" || resolvedPort == DefaultPort {
			resolvedPort = op
		}
	}

	// PORT override
	cleanP := strings.Trim(port, " \t\r\n/")
	if cleanP != "" {
		p, err := parsePortString(cleanP)
		if err != nil {
			return "", "", fmt.Errorf("PORT: %w", err)
		}
		if cleanOP == "" && (resolvedPort == "" || resolvedPort == DefaultPort) {
			resolvedPort = p
		}
	}

	if resolvedPort == "" {
		resolvedPort = DefaultPort
	}

	if host == "" {
		listenAddr = ":" + resolvedPort
	} else {
		listenAddr = host + ":" + resolvedPort
	}

	return listenAddr, resolvedPort, nil
}

// NormalizeListenAddr normalizes listen addresses across ORBIT_LISTEN, ORBIT_PORT, PORT
// into a net.Listen-compatible address. Returns ":8787" on unparseable inputs.
func NormalizeListenAddr(orbitListen, orbitPort, port string) string {
	addr, _, err := ResolveListenConfig(orbitListen, orbitPort, port)
	if err != nil {
		return DefaultListenAddr
	}
	return addr
}

// ResolvePort returns the resolved numeric port string (e.g. "8787").
func ResolvePort(orbitListen, orbitPort, port string) string {
	_, p, err := ResolveListenConfig(orbitListen, orbitPort, port)
	if err != nil {
		return DefaultPort
	}
	return p
}

// ResolveHealthcheckPort resolves the port to use for healthcheck probes from environment variables.
func ResolveHealthcheckPort() string {
	return ResolvePort(os.Getenv("ORBIT_LISTEN"), os.Getenv("ORBIT_PORT"), os.Getenv("PORT"))
}

// ResolveHealthcheckURL returns the loopback healthcheck target URL targeting /api/system/health.
func ResolveHealthcheckURL() string {
	return fmt.Sprintf("http://127.0.0.1:%s/api/system/health", ResolveHealthcheckPort())
}


