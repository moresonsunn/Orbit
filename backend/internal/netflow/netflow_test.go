package netflow

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/phili/orbit/internal/config"
	"github.com/phili/orbit/internal/docker"
)

func TestAssessPortSecurity(t *testing.T) {
	tests := []struct {
		name          string
		port          uint16
		hostIP        string
		proto         string
		detectedProto string
		expectedLevel string
		expectedBadge string
	}{
		// Critical: Unencrypted databases/caches bound to 0.0.0.0 or ::
		{
			name:          "Redis on 0.0.0.0",
			port:          6379,
			hostIP:        "0.0.0.0",
			proto:         "tcp",
			detectedProto: "Redis",
			expectedLevel: "critical",
			expectedBadge: "Critical Risk",
		},
		{
			name:          "MongoDB on ::",
			port:          27017,
			hostIP:        "::",
			proto:         "tcp6",
			detectedProto: "MongoDB",
			expectedLevel: "critical",
			expectedBadge: "Critical Risk",
		},
		{
			name:          "PostgreSQL on 0.0.0.0",
			port:          5432,
			hostIP:        "0.0.0.0",
			proto:         "tcp",
			detectedProto: "PostgreSQL",
			expectedLevel: "critical",
			expectedBadge: "Critical Risk",
		},
		{
			name:          "MySQL on 0.0.0.0",
			port:          3306,
			hostIP:        "0.0.0.0",
			proto:         "tcp",
			detectedProto: "MySQL",
			expectedLevel: "critical",
			expectedBadge: "Critical Risk",
		},
		{
			name:          "Elasticsearch on empty hostIP (treated as all-interfaces)",
			port:          9200,
			hostIP:        "",
			proto:         "tcp",
			detectedProto: "Elasticsearch",
			expectedLevel: "critical",
			expectedBadge: "Critical Risk",
		},

		// Caution: Unencrypted HTTP or unclassified ports on 0.0.0.0
		{
			name:          "HTTP 80 on 0.0.0.0",
			port:          80,
			hostIP:        "0.0.0.0",
			proto:         "tcp",
			detectedProto: "HTTP",
			expectedLevel: "caution",
			expectedBadge: "Caution",
		},
		{
			name:          "HTTP 8080 on 0.0.0.0",
			port:          8080,
			hostIP:        "0.0.0.0",
			proto:         "tcp",
			detectedProto: "HTTP",
			expectedLevel: "caution",
			expectedBadge: "Caution",
		},
		{
			name:          "Unclassified port on 0.0.0.0",
			port:          19999,
			hostIP:        "0.0.0.0",
			proto:         "tcp",
			detectedProto: "UNKNOWN",
			expectedLevel: "caution",
			expectedBadge: "Caution",
		},

		// Safe: Encrypted services or services bound to localhost
		{
			name:          "HTTPS 443 on 0.0.0.0",
			port:          443,
			hostIP:        "0.0.0.0",
			proto:         "tcp",
			detectedProto: "HTTPS",
			expectedLevel: "safe",
			expectedBadge: "Safe",
		},
		{
			name:          "SSH 22 on 0.0.0.0",
			port:          22,
			hostIP:        "0.0.0.0",
			proto:         "tcp",
			detectedProto: "SSH",
			expectedLevel: "safe",
			expectedBadge: "Safe",
		},
		{
			name:          "Redis on 127.0.0.1 loopback",
			port:          6379,
			hostIP:        "127.0.0.1",
			proto:         "tcp",
			detectedProto: "Redis",
			expectedLevel: "safe",
			expectedBadge: "Safe",
		},
		{
			name:          "HTTP on ::1 loopback",
			port:          80,
			hostIP:        "::1",
			proto:         "tcp",
			detectedProto: "HTTP",
			expectedLevel: "safe",
			expectedBadge: "Safe",
		},

		// Internal: Container-to-container private network bindings
		{
			name:          "Internal container IPv4",
			port:          80,
			hostIP:        "172.18.0.3",
			proto:         "tcp",
			detectedProto: "HTTP",
			expectedLevel: "internal",
			expectedBadge: "Internal",
		},
		{
			name:          "Internal container Redis",
			port:          6379,
			hostIP:        "10.0.0.5",
			proto:         "tcp",
			detectedProto: "Redis",
			expectedLevel: "internal",
			expectedBadge: "Internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, badge, desc := AssessPortSecurity(tt.port, tt.hostIP, tt.proto, tt.detectedProto)
			if level != tt.expectedLevel {
				t.Errorf("AssessPortSecurity level = %q, want %q", level, tt.expectedLevel)
			}
			if badge != tt.expectedBadge {
				t.Errorf("AssessPortSecurity badge = %q, want %q", badge, tt.expectedBadge)
			}
			if desc == "" {
				t.Errorf("AssessPortSecurity desc should not be empty")
			}
		})
	}
}

func TestSynthesizeContainerSockets(t *testing.T) {
	discardLog := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{HostProc: "/nonexistent-proc-path"}
	inv := docker.NewInventory(nil, discardLog)
	sr := NewSocketReader(cfg, inv)

	containers := []docker.Container{
		{
			ID:      "c1_redis_id_full",
			ShortID: "c1_redis",
			Name:    "redis-prod",
			Image:   "redis:7.2-alpine",
			Command: "redis-server --protected-mode no",
			State:   "running",
			Ports: []docker.Port{
				{
					PrivatePort: 6379,
					PublicPort:  6379,
					Protocol:    "tcp",
					HostIP:      "0.0.0.0",
				},
			},
			Networks: []docker.ContainerNet{
				{
					Name:      "bridge",
					IPAddress: "172.17.0.2",
					PrefixLen: 16,
				},
			},
		},
		{
			ID:      "c2_web_id_full",
			ShortID: "c2_web",
			Name:    "web-frontend",
			Image:   "nginx:alpine",
			Command: "nginx -g 'daemon off;'",
			State:   "running",
			Ports: []docker.Port{
				{
					PrivatePort: 80,
					PublicPort:  8080,
					Protocol:    "tcp",
					HostIP:      "127.0.0.1",
				},
			},
			ExposedPorts: []string{"443/tcp"},
			Networks: []docker.ContainerNet{
				{
					Name:      "bridge",
					IPAddress: "172.17.0.3",
					PrefixLen: 16,
				},
			},
		},
	}

	conns := sr.synthesizeContainerSockets(containers)
	if len(conns) < 3 {
		t.Fatalf("expected at least 3 synthesized connections, got %d", len(conns))
	}

	// Verify Redis on 0.0.0.0:6379 is classified critical
	foundRedisPublic := false
	for _, conn := range conns {
		if conn.ContainerID == "c1_redis_id_full" && conn.LocalPort == 6379 && conn.LocalIP == "0.0.0.0" {
			foundRedisPublic = true
			if conn.SecurityLevel != "critical" {
				t.Errorf("expected Redis public port to be critical, got %s", conn.SecurityLevel)
			}
			if conn.ProcessName != "redis-server" {
				t.Errorf("expected ProcessName 'redis-server', got %q", conn.ProcessName)
			}
			if conn.ContainerName != "redis-prod" {
				t.Errorf("expected ContainerName 'redis-prod', got %q", conn.ContainerName)
			}
			if conn.State != "LISTEN" {
				t.Errorf("expected State 'LISTEN', got %q", conn.State)
			}
		}
	}
	if !foundRedisPublic {
		t.Errorf("did not find public Redis connection in synthesized sockets")
	}

	// Verify Nginx 8080 on 127.0.0.1 is safe
	foundNginxLocal := false
	for _, conn := range conns {
		if conn.ContainerID == "c2_web_id_full" && conn.LocalPort == 8080 && conn.LocalIP == "127.0.0.1" {
			foundNginxLocal = true
			if conn.SecurityLevel != "safe" {
				t.Errorf("expected local 8080 to be safe, got %s", conn.SecurityLevel)
			}
			if conn.ProcessName != "nginx" {
				t.Errorf("expected ProcessName 'nginx', got %q", conn.ProcessName)
			}
		}
	}
	if !foundNginxLocal {
		t.Errorf("did not find local Nginx connection in synthesized sockets")
	}
}

func TestBuilderBuildSnapshotEnrichment(t *testing.T) {
	discardLog := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{HostProc: "/nonexistent-proc-path"}
	inv := docker.NewInventory(nil, discardLog)
	streamer := docker.NewStatsStreamer(nil, inv, 10, discardLog)
	builder := NewBuilder(cfg, inv, streamer)

	snapshot := builder.BuildSnapshot(context.Background())

	// Totals must exist
	if snapshot.Totals.ActiveConnections < 0 {
		t.Errorf("expected non-negative ActiveConnections, got %d", snapshot.Totals.ActiveConnections)
	}

	// Connections and ContainerTraffic slices must be non-nil
	if snapshot.Connections == nil {
		t.Errorf("expected non-nil Connections slice")
	}
	if snapshot.ContainerTraffic == nil {
		t.Errorf("expected non-nil ContainerTraffic slice")
	}
}

func TestExtractProcessName(t *testing.T) {
	if name := extractProcessName("redis-server --protected-mode no", "redis:7.2"); name != "redis-server" {
		t.Errorf("expected redis-server, got %q", name)
	}
	if name := extractProcessName("/usr/sbin/nginx -g 'daemon off;'", "nginx:alpine"); name != "nginx" {
		t.Errorf("expected nginx, got %q", name)
	}
	if name := extractProcessName("", "docker.io/library/postgres:16"); name != "postgres" {
		t.Errorf("expected postgres, got %q", name)
	}
	if name := extractProcessName("", ""); name != "app" {
		t.Errorf("expected app, got %q", name)
	}
}
