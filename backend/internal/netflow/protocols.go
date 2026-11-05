package netflow

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
	3306:  "MySQL",
	5432:  "PostgreSQL",
	6379:  "Redis",
	8080:  "HTTP",
	8443:  "HTTPS",
	8787:  "Orbit",
	9000:  "Portainer",
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

