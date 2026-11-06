package netflow

import (
	"fmt"
	"net"
	"strings"
)

var wellKnownPorts = map[uint16]string{
	21:    "FTP",
	22:    "SSH",
	25:    "SMTP",
	53:    "DNS",
	67:    "DHCP",
	68:    "DHCP",
	80:    "HTTP",
	123:   "NTP",
	443:   "HTTPS",
	1883:  "MQTT",
	2375:  "Docker Daemon",
	2376:  "Docker TLS",
	3000:  "Grafana",
	3306:  "MySQL",
	5432:  "PostgreSQL",
	5672:  "RabbitMQ",
	6379:  "Redis",
	6443:  "Kubernetes",
	8000:  "HTTP",
	8080:  "HTTP",
	8443:  "HTTPS",
	8787:  "Orbit",
	8888:  "HTTP",
	9000:  "Portainer",
	9090:  "Prometheus",
	9200:  "Elasticsearch",
	11211: "Memcached",
	27017: "MongoDB",
}

// ClassifyPort identifies the likely application protocol based on the port number.
// In accordance with Master Prompt Section 7.3, this classification is explicitly
// tagged as "port_based" rather than deep packet inspection.
func ClassifyPort(port uint16) string {
	if name, ok := wellKnownPorts[port]; ok {
		return name
	}
	return "UNKNOWN"
}

// AssessPortSecurity evaluates the security risk of a listening or connected port endpoint.
// Levels:
// - "critical": Unencrypted databases/caches (Redis, Mongo, Postgres, MySQL) bound to public 0.0.0.0 or ::
// - "caution": Unencrypted HTTP (80, 8080) or unclassified ports bound to public 0.0.0.0 or ::
// - "safe": Encrypted services (HTTPS 443, SSH 22) or services bound to localhost (127.0.0.1, ::1)
// - "internal": Container-to-container private network bindings (e.g. 172.17.x.x, 10.x.x.x, 192.168.x.x)
func AssessPortSecurity(port uint16, hostIP string, proto string, detectedProto string) (level string, badge string, desc string) {
	cleanIP := strings.TrimSpace(hostIP)

	// 1. Localhost loopback binding is always safe
	if cleanIP == "127.0.0.1" || cleanIP == "::1" || cleanIP == "localhost" || strings.HasPrefix(cleanIP, "127.") {
		return "safe", "Safe", fmt.Sprintf("Service bound to localhost loopback (%s)", cleanIP)
	}

	parsedIP := net.ParseIP(cleanIP)

	// 2. Container-to-container private network binding (RFC 1918 / ULA)
	if parsedIP != nil && (parsedIP.IsPrivate() || parsedIP.IsLinkLocalUnicast()) {
		return "internal", "Internal", fmt.Sprintf("Container private network endpoint (%s)", cleanIP)
	}

	// 3. Public or non-loopback exposure (0.0.0.0, ::, empty, or routable host IP)
	displayIP := cleanIP
	if displayIP == "" || displayIP == "0.0.0.0" || displayIP == "::" || displayIP == "*" {
		displayIP = "0.0.0.0"
	}

	// Check if encrypted
	isEncrypted := port == 443 || port == 8443 || port == 22 || port == 2376 ||
		strings.EqualFold(detectedProto, "HTTPS") ||
		strings.EqualFold(detectedProto, "SSH") ||
		strings.EqualFold(detectedProto, "Docker TLS")
	if isEncrypted {
		return "safe", "Safe", fmt.Sprintf("Encrypted service (%s) accessible on %s", detectedProto, displayIP)
	}

	// Check if unencrypted critical database/cache
	isCriticalDB := port == 6379 || port == 27017 || port == 5432 || port == 3306 ||
		port == 11211 || port == 9200 || port == 2375 ||
		strings.EqualFold(detectedProto, "Redis") ||
		strings.EqualFold(detectedProto, "MongoDB") ||
		strings.EqualFold(detectedProto, "PostgreSQL") ||
		strings.EqualFold(detectedProto, "MySQL") ||
		strings.EqualFold(detectedProto, "Memcached") ||
		strings.EqualFold(detectedProto, "Elasticsearch") ||
		strings.EqualFold(detectedProto, "Docker Daemon")
	if isCriticalDB {
		return "critical", "Critical Risk", fmt.Sprintf("Unencrypted database/cache (%s) exposed publicly on %s", detectedProto, displayIP)
	}

	// Check if unencrypted HTTP
	isHTTP := port == 80 || port == 8080 || port == 8000 || port == 8888 ||
		strings.EqualFold(detectedProto, "HTTP")
	if isHTTP {
		return "caution", "Caution", fmt.Sprintf("Unencrypted HTTP service exposed publicly on %s", displayIP)
	}

	// Unclassified or other ports bound publicly
	protoName := detectedProto
	if protoName == "" || protoName == "UNKNOWN" {
		protoName = fmt.Sprintf("port %d", port)
	}
	return "caution", "Caution", fmt.Sprintf("Unclassified service (%s) exposed publicly on %s", protoName, displayIP)
}

