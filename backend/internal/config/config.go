// Package config loads Orbit's static configuration from environment variables.
//
// Runtime-adjustable settings (intervals, retention, alert thresholds) live in the
// database (see package settings); this package only covers values that must be
// known at process start or that are security relevant and therefore must not be
// changeable from the web UI (e.g. enabling management actions).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
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
	c := &Config{
		Listen:            env("ORBIT_LISTEN", ":8787"),
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
	var err error
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

