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
		allConnections = append(allConnections, conns...)
	}

	// Also read host sockets
	hostNetDir := filepath.Join(sr.cfg.HostProc, "net")
	if fi, err := os.Stat(hostNetDir); err != nil || !fi.IsDir() {
		hostNetDir = "/proc/net"
	}
	hostConns := sr.readFromDir(hostNetDir, "host", "Host-System", byIP)
	allConnections = append(allConnections, hostConns...)

	return allConnections
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

