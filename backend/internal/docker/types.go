package docker

import (
	"sort"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
)

// Container is Orbit's normalized view of a Docker container.
type Container struct {
	ID             string            `json:"id"`
	ShortID        string            `json:"shortId"`
	Name           string            `json:"name"`
	Image          string            `json:"image"`
	ImageRepo      string            `json:"imageRepo"`
	ImageTag       string            `json:"imageTag"`
	ImageID        string            `json:"imageId"`
	Command        string            `json:"command"`
	State          string            `json:"state"`  // created|running|paused|restarting|removing|exited|dead
	Health         string            `json:"health"` // healthy|unhealthy|starting|none
	HealthStreak   int               `json:"healthFailingStreak"`
	Created        time.Time         `json:"created"`
	StartedAt      *time.Time        `json:"startedAt"`
	FinishedAt     *time.Time        `json:"finishedAt"`
	RestartCount   int               `json:"restartCount"`
	ExitCode       int               `json:"exitCode"`
	OOMKilled      bool              `json:"oomKilled"`
	Error          string            `json:"error,omitempty"`
	RestartPolicy  string            `json:"restartPolicy"`
	MaxRetry       int               `json:"restartMaxRetry"`
	ComposeProject string            `json:"composeProject,omitempty"`
	ComposeService string            `json:"composeService,omitempty"`
	AppGroup       string            `json:"appGroup"`
	AppSource      string            `json:"appSource"` // compose|casaos|standalone
	Labels         map[string]string `json:"labels"`
	Ports          []Port            `json:"ports"`
	ExposedPorts   []string          `json:"exposedPorts"`
	NetworkMode    string            `json:"networkMode"`
	Networks       []ContainerNet    `json:"networks"`
	Mounts         []Mount           `json:"mounts"`
	Limits         Limits            `json:"limits"`
	Privileged     bool              `json:"privileged"`
	Pid            int               `json:"-"`
	SandboxKey     string            `json:"-"`
	Runtime        string            `json:"runtime,omitempty"`
	HasGPURequest  bool              `json:"gpuRequested"`
	Healthcheck    bool              `json:"hasHealthcheck"`
	IsOrbit        bool              `json:"isOrbit"`
}

type Port struct {
	PrivatePort uint16 `json:"privatePort"`
	PublicPort  uint16 `json:"publicPort,omitempty"`
	Protocol    string `json:"protocol"`
	HostIP      string `json:"hostIp,omitempty"`
}

type ContainerNet struct {
	Name       string   `json:"name"`
	NetworkID  string   `json:"networkId"`
	EndpointID string   `json:"endpointId"`
	IPAddress  string   `json:"ipAddress,omitempty"`
	PrefixLen  int      `json:"prefixLen,omitempty"`
	IPv6       string   `json:"ipv6Address,omitempty"`
	Gateway    string   `json:"gateway,omitempty"`
	MacAddress string   `json:"macAddress,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
}

type Mount struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	RW          bool   `json:"rw"`
	Driver      string `json:"driver,omitempty"`
}

type Limits struct {
	MemoryBytes int64   `json:"memoryBytes"` // 0 = unlimited
	MemorySwap  int64   `json:"memorySwapBytes"`
	CPUs        float64 `json:"cpus"` // 0 = unlimited
	CPUShares   int64   `json:"cpuShares"`
	CpusetCPUs  string  `json:"cpusetCpus,omitempty"`
	PidsLimit   int64   `json:"pidsLimit"`
}

func parseDockerTime(s string) *time.Time {
	if s == "" || strings.HasPrefix(s, "0001-01-01") {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return nil
	}
	return &t
}

// SplitImage splits an image reference into repository and tag.
func SplitImage(ref string) (repo, tag string) {
	if i := strings.Index(ref, "@"); i >= 0 {
		return ref[:i], ref[i+1:]
	}
	slash := strings.LastIndex(ref, "/")
	if i := strings.LastIndex(ref, ":"); i > slash {
		return ref[:i], ref[i+1:]
	}
	if strings.HasPrefix(ref, "sha256:") {
		return ref, ""
	}
	return ref, "latest"
}

// FromInspect converts an inspect response into the Orbit model.
func FromInspect(in container.InspectResponse) Container {
	c := Container{
		ID:      in.ID,
		ShortID: shortID(in.ID),
		Name:    strings.TrimPrefix(in.Name, "/"),
		ImageID: in.Image,
		Labels:  map[string]string{},
		Runtime: "",
	}
	if t := parseDockerTime(in.Created); t != nil {
		c.Created = *t
	}
	c.RestartCount = in.RestartCount
	if in.Config != nil {
		c.Image = in.Config.Image
		if in.Config.Labels != nil {
			c.Labels = in.Config.Labels
		}
		c.Healthcheck = in.Config.Healthcheck != nil && len(in.Config.Healthcheck.Test) > 0 && in.Config.Healthcheck.Test[0] != "NONE"
		cmd := append(append([]string{}, in.Config.Entrypoint...), in.Config.Cmd...)
		c.Command = strings.Join(cmd, " ")
		for p := range in.Config.ExposedPorts {
			c.ExposedPorts = append(c.ExposedPorts, p.String())
		}
		sort.Strings(c.ExposedPorts)
	}
	c.ImageRepo, c.ImageTag = SplitImage(c.Image)
	if st := in.State; st != nil {
		c.State = string(st.Status)
		c.Pid = st.Pid
		c.ExitCode = st.ExitCode
		c.OOMKilled = st.OOMKilled
		c.Error = st.Error
		c.StartedAt = parseDockerTime(st.StartedAt)
		c.FinishedAt = parseDockerTime(st.FinishedAt)
		c.Health = "none"
		if st.Health != nil && st.Health.Status != "" {
			c.Health = string(st.Health.Status)
			c.HealthStreak = st.Health.FailingStreak
		}
	}
	if hc := in.HostConfig; hc != nil {
		c.NetworkMode = string(hc.NetworkMode)
		c.RestartPolicy = string(hc.RestartPolicy.Name)
		if c.RestartPolicy == "" {
			c.RestartPolicy = "no"
		}
		c.MaxRetry = hc.RestartPolicy.MaximumRetryCount
		c.Privileged = hc.Privileged
		c.Runtime = hc.Runtime
		c.Limits = Limits{MemoryBytes: hc.Memory, MemorySwap: hc.MemorySwap, CPUShares: hc.CPUShares, CpusetCPUs: hc.CpusetCpus}
		if hc.NanoCPUs > 0 {
			c.Limits.CPUs = float64(hc.NanoCPUs) / 1e9
		} else if hc.CPUQuota > 0 && hc.CPUPeriod > 0 {
			c.Limits.CPUs = float64(hc.CPUQuota) / float64(hc.CPUPeriod)
		}
		if hc.PidsLimit != nil {
			c.Limits.PidsLimit = *hc.PidsLimit
		}
		for _, dr := range hc.DeviceRequests {
			for _, caps := range dr.Capabilities {
				for _, cp := range caps {
					if cp == "gpu" {
						c.HasGPURequest = true
					}
				}
			}
			if dr.Driver == "nvidia" {
				c.HasGPURequest = true
			}
		}
		if c.Runtime == "nvidia" {
			c.HasGPURequest = true
		}
	}
	if ns := in.NetworkSettings; ns != nil {
		c.SandboxKey = ns.SandboxKey
		for p, binds := range ns.Ports {
			if len(binds) == 0 {
				continue
			}
			for _, b := range binds {
				hp := parsePort(b.HostPort)
				pp := Port{PrivatePort: p.Num(), PublicPort: hp, Protocol: string(p.Proto())}
				if b.HostIP.IsValid() {
					pp.HostIP = b.HostIP.String()
				}
				c.Ports = append(c.Ports, pp)
			}
		}
		for name, ep := range ns.Networks {
			if ep == nil {
				continue
			}
			cn := ContainerNet{Name: name, NetworkID: ep.NetworkID, EndpointID: ep.EndpointID, PrefixLen: ep.IPPrefixLen, Aliases: ep.Aliases}
			if ep.IPAddress.IsValid() {
				cn.IPAddress = ep.IPAddress.String()
			}
			if ep.GlobalIPv6Address.IsValid() {
				cn.IPv6 = ep.GlobalIPv6Address.String()
			}
			if ep.Gateway.IsValid() {
				cn.Gateway = ep.Gateway.String()
			}
			cn.MacAddress = ep.MacAddress.String()
			c.Networks = append(c.Networks, cn)
		}
		sort.Slice(c.Networks, func(i, j int) bool { return c.Networks[i].Name < c.Networks[j].Name })
	}
	sort.Slice(c.Ports, func(i, j int) bool {
		if c.Ports[i].PrivatePort != c.Ports[j].PrivatePort {
			return c.Ports[i].PrivatePort < c.Ports[j].PrivatePort
		}
		return c.Ports[i].HostIP < c.Ports[j].HostIP
	})
	for _, m := range in.Mounts {
		c.Mounts = append(c.Mounts, Mount{Type: string(m.Type), Name: m.Name, Source: m.Source, Destination: m.Destination, RW: m.RW, Driver: m.Driver})
	}
	c.ComposeProject = c.Labels["com.docker.compose.project"]
	c.ComposeService = c.Labels["com.docker.compose.service"]
	c.AppGroup, c.AppSource = appGroup(c)
	c.IsOrbit = c.Labels["org.opencontainers.image.title"] == "orbit" || c.Labels["dev.orbit.self"] == "true"
	if c.Ports == nil {
		c.Ports = []Port{}
	}
	if c.Networks == nil {
		c.Networks = []ContainerNet{}
	}
	if c.Mounts == nil {
		c.Mounts = []Mount{}
	}
	if c.ExposedPorts == nil {
		c.ExposedPorts = []string{}
	}
	return c
}

// appGroup derives a logical application group. CasaOS apps are compose projects
// whose compose file carries x-casaos metadata; Docker does not expose that
// metadata as labels, so we only use labels that are actually present.
func appGroup(c Container) (group, source string) {
	for _, k := range []string{"io.casaos.v1.app.store.id", "casaos.app", "io.casaos.app"} {
		if v := c.Labels[k]; v != "" {
			return v, "casaos"
		}
	}
	if c.ComposeProject != "" {
		if _, ok := c.Labels["icon"]; ok { // CasaOS sets an "icon" label on its compose apps
			return c.ComposeProject, "casaos"
		}
		return c.ComposeProject, "compose"
	}
	return "", "standalone"
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func parsePort(s string) uint16 {
	var n int
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
		if n > 65535 {
			return 0
		}
	}
	return uint16(n)
}

