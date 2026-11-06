package netflow

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/phili/orbit/internal/config"
	"github.com/phili/orbit/internal/docker"
)

var tcpStates = map[string]string{
	"01": "ESTABLISHED",
	"02": "SYN_SENT",
	"03": "SYN_RECV",
	"04": "FIN_WAIT1",
	"05": "FIN_WAIT2",
	"06": "TIME_WAIT",
	"07": "CLOSE",
	"08": "CLOSE_WAIT",
	"09": "LAST_ACK",
	"0A": "LISTEN",
	"0B": "CLOSING",
}

type SocketReader struct {
	cfg *config.Config
	inv *docker.Inventory
}

func NewSocketReader(cfg *config.Config, inv *docker.Inventory) *SocketReader {
	return &SocketReader{
		cfg: cfg,
		inv: inv,
	}
}

func (sr *SocketReader) ReadSockets() []SocketConnection {
	containers := sr.inv.Containers()
	byIP := sr.inv.ByIP()
	var allConnections []SocketConnection

	containerSocketsCount := 0

	// Read sockets for running containers via their container PID namespace
	for _, c := range containers {
		if c.State != "running" || c.Pid <= 0 {
			continue
		}

		pidNetDir := filepath.Join(sr.cfg.HostProc, fmt.Sprintf("%d", c.Pid), "net")
		if fi, err := os.Stat(pidNetDir); err != nil || !fi.IsDir() {
			pidNetDir = filepath.Join("/proc", fmt.Sprintf("%d", c.Pid), "net")
			if fi, err := os.Stat(pidNetDir); err != nil || !fi.IsDir() {
				continue
			}
		}

		conns := sr.readFromDir(pidNetDir, c.ID, c.Name, byIP)
		containerSocketsCount += len(conns)
		allConnections = append(allConnections, conns...)
	}

	// Also read host sockets
	hostNetDir := filepath.Join(sr.cfg.HostProc, "net")
	if fi, err := os.Stat(hostNetDir); err != nil || !fi.IsDir() {
		hostNetDir = "/proc/net"
	}
	hostConns := sr.readFromDir(hostNetDir, "host", "Host-System", byIP)
	allConnections = append(allConnections, hostConns...)

	// If procfs yielded 0 container sockets (e.g. missing /host/proc in orbit-live or Docker Desktop),
	// synthesize container listening endpoints from Docker container data (c.Ports and c.ExposedPorts)
	if containerSocketsCount == 0 {
		synthConns := sr.synthesizeContainerSockets(containers)
		allConnections = append(allConnections, synthConns...)
	}

	// Enrich connections with ProcessName, SecurityLevel, SecurityBadge, SecurityDesc, and deduplicate
	return sr.enrichAndDeduplicate(allConnections, containers)
}

func (sr *SocketReader) synthesizeContainerSockets(containers []docker.Container) []SocketConnection {
	var results []SocketConnection

	for _, c := range containers {
		if c.State != "running" {
			continue
		}

		procName := extractProcessName(c.Command, c.Image)

		// Preferred container IP from network attachments
		containerIP := "127.0.0.1"
		for _, netRef := range c.Networks {
			if netRef.IPAddress != "" {
				containerIP = netRef.IPAddress
				break
			}
		}

		seenPorts := make(map[string]bool)

		// 1. Process container port mappings (c.Ports)
		for _, p := range c.Ports {
			proto := strings.ToLower(p.Protocol)
			if proto == "" {
				proto = "tcp"
			}

			appProto := ClassifyPort(p.PrivatePort)

			// If published to host (PublicPort > 0)
			if p.PublicPort > 0 {
				hostIP := p.HostIP
				if hostIP == "" {
					hostIP = "0.0.0.0"
				}

				secLevel, secBadge, secDesc := AssessPortSecurity(p.PublicPort, hostIP, proto, appProto)

				results = append(results, SocketConnection{
					Protocol:       proto,
					LocalIP:        hostIP,
					LocalPort:      p.PublicPort,
					LocalAddr:      hostIP,
					RemoteIP:       "0.0.0.0",
					RemotePort:     0,
					RemoteAddr:     "0.0.0.0",
					State:          "LISTEN",
					ContainerID:    c.ID,
					ContainerName:  c.Name,
					ProcessName:    procName,
					AppProtocol:    appProto,
					DetectedProto:  appProto,
					Classification: "port_based",
					DetectionMode:  "docker_engine",
					SecurityLevel:  secLevel,
					SecurityBadge:  secBadge,
					SecurityDesc:   secDesc,
				})
			}

			// Also synthesize internal listening port
			portKey := fmt.Sprintf("%s:%d", proto, p.PrivatePort)
			if !seenPorts[portKey] {
				seenPorts[portKey] = true
				secLevel, secBadge, secDesc := AssessPortSecurity(p.PrivatePort, containerIP, proto, appProto)

				results = append(results, SocketConnection{
					Protocol:       proto,
					LocalIP:        containerIP,
					LocalPort:      p.PrivatePort,
					LocalAddr:      containerIP,
					RemoteIP:       "0.0.0.0",
					RemotePort:     0,
					RemoteAddr:     "0.0.0.0",
					State:          "LISTEN",
					ContainerID:    c.ID,
					ContainerName:  c.Name,
					ProcessName:    procName,
					AppProtocol:    appProto,
					DetectedProto:  appProto,
					Classification: "port_based",
					DetectionMode:  "docker_engine",
					SecurityLevel:  secLevel,
					SecurityBadge:  secBadge,
					SecurityDesc:   secDesc,
				})
			}
		}

		// 2. Process exposed ports not published
		for _, exp := range c.ExposedPorts {
			parts := strings.Split(exp, "/")
			if len(parts) == 0 {
				continue
			}
			portNum, err := strconv.ParseUint(parts[0], 10, 16)
			if err != nil {
				continue
			}
			proto := "tcp"
			if len(parts) > 1 && parts[1] != "" {
				proto = strings.ToLower(parts[1])
			}

			portKey := fmt.Sprintf("%s:%d", proto, portNum)
			if seenPorts[portKey] {
				continue
			}
			seenPorts[portKey] = true

			appProto := ClassifyPort(uint16(portNum))
			secLevel, secBadge, secDesc := AssessPortSecurity(uint16(portNum), containerIP, proto, appProto)

			results = append(results, SocketConnection{
				Protocol:       proto,
				LocalIP:        containerIP,
				LocalPort:      uint16(portNum),
				LocalAddr:      containerIP,
				RemoteIP:       "0.0.0.0",
				RemotePort:     0,
				RemoteAddr:     "0.0.0.0",
				State:          "LISTEN",
				ContainerID:    c.ID,
				ContainerName:  c.Name,
				ProcessName:    procName,
				AppProtocol:    appProto,
				DetectedProto:  appProto,
				Classification: "port_based",
				DetectionMode:  "docker_engine",
				SecurityLevel:  secLevel,
				SecurityBadge:  secBadge,
				SecurityDesc:   secDesc,
			})
		}
	}

	return results
}

func extractProcessName(command, image string) string {
	if command != "" {
		fields := strings.Fields(command)
		if len(fields) > 0 {
			cmd := strings.Trim(fields[0], "\"'")
			base := filepath.Base(cmd)
			if base != "" && base != "." && base != "/" {
				return base
			}
		}
	}
	if image != "" {
		base := image
		if idx := strings.LastIndex(base, "/"); idx != -1 {
			base = base[idx+1:]
		}
		if idx := strings.Index(base, ":"); idx != -1 {
			base = base[:idx]
		}
		if base != "" {
			return base
		}
	}
	return "app"
}

func (sr *SocketReader) enrichAndDeduplicate(conns []SocketConnection, containers []docker.Container) []SocketConnection {
	deduped := []SocketConnection{}
	seen := make(map[string]bool)

	cMap := make(map[string]docker.Container, len(containers))
	for _, c := range containers {
		cMap[c.ID] = c
	}

	for _, conn := range conns {
		// Populate compatibility addresses
		if conn.LocalAddr == "" {
			conn.LocalAddr = conn.LocalIP
		}
		if conn.RemoteAddr == "" {
			conn.RemoteAddr = conn.RemoteIP
		}

		// Lookup container attribution if missing
		if conn.ContainerID != "" && conn.ContainerID != "host" {
			if c, ok := cMap[conn.ContainerID]; ok {
				if conn.ContainerName == "" {
					conn.ContainerName = c.Name
				}
				if conn.ProcessName == "" {
					conn.ProcessName = extractProcessName(c.Command, c.Image)
				}
			}
		} else if conn.ContainerID == "host" {
			if conn.ProcessName == "" {
				conn.ProcessName = "system"
			}
		}

		// Classification & application protocol
		if conn.AppProtocol == "" || conn.AppProtocol == "UNKNOWN" {
			conn.AppProtocol = ClassifyPort(conn.LocalPort)
			if conn.AppProtocol == "UNKNOWN" {
				conn.AppProtocol = ClassifyPort(conn.RemotePort)
			}
		}
		if conn.DetectedProto == "" {
			conn.DetectedProto = conn.AppProtocol
		}

		// Security assessment
		if conn.SecurityLevel == "" {
			conn.SecurityLevel, conn.SecurityBadge, conn.SecurityDesc = AssessPortSecurity(conn.LocalPort, conn.LocalIP, conn.Protocol, conn.AppProtocol)
		}

		// Deduplication key
		key := fmt.Sprintf("%s:%s:%d:%s:%d:%s", conn.Protocol, conn.LocalIP, conn.LocalPort, conn.RemoteIP, conn.RemotePort, conn.ContainerID)
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, conn)
	}

	return deduped
}

func (sr *SocketReader) readFromDir(netDir, defaultContainerID, defaultContainerName string, byIP map[string]docker.Container) []SocketConnection {
	var results []SocketConnection

	files := []struct {
		name  string
		proto string
		isTCP bool
	}{
		{"tcp", "tcp", true},
		{"tcp6", "tcp6", true},
		{"udp", "udp", false},
		{"udp6", "udp6", false},
	}

	for _, f := range files {
		path := filepath.Join(netDir, f.name)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		// Skip header line
		if scanner.Scan() {
			// header
		}

		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 4 {
				continue
			}

			localIP, localPort, ok := parseAddr(fields[1], strings.HasSuffix(f.name, "6"))
			if !ok {
				continue
			}
			remoteIP, remotePort, ok := parseAddr(fields[2], strings.HasSuffix(f.name, "6"))
			if !ok {
				continue
			}

			state := "ACTIVE"
			if f.isTCP {
				stCode := strings.ToUpper(fields[3])
				if s, exists := tcpStates[stCode]; exists {
					state = s
				} else {
					state = stCode
				}
			}

			cID := defaultContainerID
			cName := defaultContainerName

			// Attempt to resolve target container by local or remote IP
			if match, exists := byIP[localIP]; exists {
				cID = match.ID
				cName = match.Name
			} else if match, exists := byIP[remoteIP]; exists {
				cID = match.ID
				cName = match.Name
			}

			// Determine application protocol (port-based)
			appProto := ClassifyPort(localPort)
			if appProto == "UNKNOWN" {
				appProto = ClassifyPort(remotePort)
			}

			conn := SocketConnection{
				Protocol:       f.proto,
				LocalIP:        localIP,
				LocalPort:      localPort,
				RemoteIP:       remoteIP,
				RemotePort:     remotePort,
				State:          state,
				ContainerID:    cID,
				ContainerName:  cName,
				AppProtocol:    appProto,
				Classification: "port_based",
				DetectionMode:  "socket_snapshot",
			}

			results = append(results, conn)
		}
	}

	return results
}

func parseAddr(hexAddr string, isV6 bool) (string, uint16, bool) {
	parts := strings.Split(hexAddr, ":")
	if len(parts) != 2 {
		return "", 0, false
	}

	portNum, err := strconv.ParseUint(parts[1], 16, 16)
	if err != nil {
		return "", 0, false
	}

	if !isV6 {
		// IPv4: 8 hex chars little endian
		if len(parts[0]) != 8 {
			return "", 0, false
		}
		ipBytes, err := hex.DecodeString(parts[0])
		if err != nil || len(ipBytes) != 4 {
			return "", 0, false
		}
		// Little-endian order in /proc/net/tcp
		ip := net.IPv4(ipBytes[3], ipBytes[2], ipBytes[1], ipBytes[0])
		return ip.String(), uint16(portNum), true
	}

	// IPv6: 32 hex chars
	if len(parts[0]) != 32 {
		return "", 0, false
	}
	raw, err := hex.DecodeString(parts[0])
	if err != nil || len(raw) != 16 {
		return "", 0, false
	}
	// Swap 4-byte words for little-endian Linux representation
	v6 := make([]byte, 16)
	for i := 0; i < 4; i++ {
		binary.BigEndian.PutUint32(v6[i*4:], binary.LittleEndian.Uint32(raw[i*4:]))
	}
	ip := net.IP(v6)
	return ip.String(), uint16(portNum), true
}

