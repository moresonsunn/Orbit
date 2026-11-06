package e2e

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// FindRepoRoot discovers the absolute path to the Orbit repository root.
func FindRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := wd
	for i := 0; i < 6; i++ {
		backendPath := filepath.Join(dir, "backend")
		frontendPath := filepath.Join(dir, "frontend")
		if stat1, err1 := os.Stat(backendPath); err1 == nil && stat1.IsDir() {
			if stat2, err2 := os.Stat(frontendPath); err2 == nil && stat2.IsDir() {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not find Orbit repository root from %s", wd)
}

// ResolveHealthcheckTargetURL computes the exact HTTP URL that Orbit's
// -healthcheck flag queries for a given ORBIT_LISTEN value.
// Specifications:
// - Default (empty ORBIT_LISTEN): http://127.0.0.1:8787/api/v1/health
// - Custom port ":8080": http://127.0.0.1:8080/api/v1/health
// - Full host:port "0.0.0.0:9090" or "127.0.0.1:9090": http://127.0.0.1:9090/api/v1/health
// - IPv6 "[::]:8080": http://127.0.0.1:8080/api/v1/health
// - Bare port "8080": http://127.0.0.1:8080/api/v1/health
func ResolveHealthcheckTargetURL(listenAddr string) (string, error) {
	clean := strings.TrimRight(strings.TrimSpace(listenAddr), "/")
	if clean == "" {
		return "http://127.0.0.1:8787/api/v1/health", nil
	}

	var port string

	// Check bare numeric port (e.g., "8080")
	if num, err := strconv.Atoi(clean); err == nil && num > 0 && num <= 65535 {
		port = strconv.Itoa(num)
	} else if strings.HasPrefix(clean, ":") {
		// Form ":8080"
		candidate := strings.TrimPrefix(clean, ":")
		if num, err := strconv.Atoi(candidate); err == nil && num > 0 && num <= 65535 {
			port = candidate
		} else {
			return "", fmt.Errorf("invalid port in listen address: %s", clean)
		}
	} else {
		// Try net.SplitHostPort (covers "0.0.0.0:8080", "[::]:8080", etc.)
		_, p, err := net.SplitHostPort(clean)
		if err == nil {
			if num, err := strconv.Atoi(p); err == nil && num > 0 && num <= 65535 {
				port = p
			} else {
				return "", fmt.Errorf("invalid port parsed: %s", p)
			}
		} else {
			// Fallback: look for last colon
			lastColon := strings.LastIndex(clean, ":")
			if lastColon != -1 && lastColon+1 < len(clean) {
				candidate := clean[lastColon+1:]
				candidate = strings.TrimRight(candidate, "/]")
				if num, err := strconv.Atoi(candidate); err == nil && num > 0 && num <= 65535 {
					port = candidate
				}
			}
		}
	}

	if port == "" {
		// If unparsable, default fallback to 8787
		port = "8787"
	}

	return fmt.Sprintf("http://127.0.0.1:%s/api/v1/health", port), nil
}

// ExecuteHealthcheckSimulation performs an HTTP GET request to the target URL
// with a 2-second timeout, mimicking backend/cmd/orbit/main.go healthcheck behavior.
// Returns exit code 0 if 200 OK, 1 otherwise.
func ExecuteHealthcheckSimulation(targetURL string) int {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(targetURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return 1
	}
	_ = resp.Body.Close()
	return 0
}

// ReadFileIfExists reads a file from disk if it exists; returns content, found boolean, and error.
func ReadFileIfExists(path string) (string, bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}
	return string(content), true, nil
}

// ParseSimpleYAMLToLines parses a YAML file into line-by-line trimmed tokens and key-value pairs.
func ParseSimpleYAMLToLines(content string) []string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

// CheckMarkdownContainsLink checks if markdown content contains a link or image matching target.
func CheckMarkdownContainsLink(content, target string) bool {
	return strings.Contains(content, target)
}

// Reference Specifications from PROJECT.md & Survey Handoffs

// ReferenceSidebarContract holds the authoritative layout properties from PROJECT.md § 1
var ReferenceSidebarContract = struct {
	CollapsedWidthClass       string
	ExpandedWidthClass        string
	CollapsedWidthPixels      int
	ExpandedWidthPixels       int
	LogoBoxPixels             int
	LogoCenterAxisX           int
	FloatingToggleClassTokens []string
	NavIconCenterAxisX        int
	NavCollapsedClassToken    string
	NavExpandedClassToken     string
}{
	CollapsedWidthClass:  "w-16",
	ExpandedWidthClass:   "w-56",
	CollapsedWidthPixels: 64,
	ExpandedWidthPixels:  224,
	LogoBoxPixels:        28,
	LogoCenterAxisX:      32,
	FloatingToggleClassTokens: []string{
		"absolute",
		"-right-3",
		"top-1/2",
		"-translate-y-1/2",
		"w-6",
		"h-6",
		"rounded-full",
		"bg-[#1c1c1c]",
		"border",
		"border-[#2e2e2e]",
		"shadow-md",
		"z-40",
	},
	NavIconCenterAxisX:     32,
	NavCollapsedClassToken: "justify-center px-0",
	NavExpandedClassToken:  "px-2.5",
}

// Reference13Views defines the authoritative inventory of Orbit views
var Reference13Views = []struct {
	Name        string
	Path        string
	Component   string
	RequiredJSX []string
}{
	{
		Name:      "Dashboard",
		Path:      "/",
		Component: "frontend/src/pages/Dashboard.tsx",
		RequiredJSX: []string{
			"showConfigModal",
			"showContainers",
			"showSummary",
			"showDockerOverview",
			"showThermals",
			"showLoadAvg",
			"showGPU",
			"showCharts",
			"compactMode",
		},
	},
	{
		Name:      "Containers",
		Path:      "/containers",
		Component: "frontend/src/pages/Containers.tsx",
		RequiredJSX: []string{
			"ContainersPage",
			"containers",
		},
	},
	{
		Name:      "Container Detail",
		Path:      "/containers/:id",
		Component: "frontend/src/pages/ContainerDetail.tsx",
		RequiredJSX: []string{
			"overflow-x-auto",
		},
	},
	{
		Name:      "Network Flow",
		Path:      "/network-flow",
		Component: "frontend/src/pages/NetworkFlow.tsx",
		RequiredJSX: []string{
			"NetworkFlowPage",
		},
	},
	{
		Name:      "Networks",
		Path:      "/networks",
		Component: "frontend/src/pages/Networks.tsx",
		RequiredJSX: []string{
			"NetworksPage",
		},
	},
	{
		Name:      "Images",
		Path:      "/images",
		Component: "frontend/src/pages/Images.tsx",
		RequiredJSX: []string{
			"ImagesPage",
		},
	},
	{
		Name:      "Volumes",
		Path:      "/volumes",
		Component: "frontend/src/pages/Volumes.tsx",
		RequiredJSX: []string{
			"VolumesPage",
		},
	},
	{
		Name:      "GPU Monitoring",
		Path:      "/gpu",
		Component: "frontend/src/pages/GPU.tsx",
		RequiredJSX: []string{
			"GPUPage",
		},
	},
	{
		Name:      "Events",
		Path:      "/events",
		Component: "frontend/src/pages/Events.tsx",
		RequiredJSX: []string{
			"EventsPage",
		},
	},
	{
		Name:      "Logs",
		Path:      "/logs",
		Component: "frontend/src/pages/Logs.tsx",
		RequiredJSX: []string{
			"LogsPage",
		},
	},
	{
		Name:      "Settings",
		Path:      "/settings",
		Component: "frontend/src/pages/Settings.tsx",
		RequiredJSX: []string{
			"SettingsPage",
		},
	},
	{
		Name:      "Login Modal",
		Path:      "/login",
		Component: "frontend/src/components/auth/LoginModal.tsx",
		RequiredJSX: []string{
			"backdrop-blur-sm",
		},
	},
	{
		Name:      "Setup Modal",
		Path:      "/setup",
		Component: "frontend/src/components/auth/SetupModal.tsx",
		RequiredJSX: []string{
			"backdrop-blur-sm",
		},
	},
}

// ReferenceCIWorkflowSpec defines required tokens for .github/workflows/ci.yml
var ReferenceCIWorkflowSpec = struct {
	Triggers         []string
	Runner           string
	GoAction         string
	NodeAction       string
	GoTestCommand    string
	GoVetCommand     string
	NpmCiCommand     string
	NpmBuildCommand  string
	GoVersionTarget  string
	NodeVersionTarget string
}{
	Triggers:          []string{"push", "pull_request", "main", "master"},
	Runner:            "ubuntu-latest",
	GoAction:          "actions/setup-go@v5",
	NodeAction:        "actions/setup-node@v4",
	GoTestCommand:     "go test -v ./...",
	GoVetCommand:      "go vet ./...",
	NpmCiCommand:      "npm ci",
	NpmBuildCommand:   "npm run build",
	GoVersionTarget:   "1.26",
	NodeVersionTarget: "22",
}

// ReferenceDockerPublishSpec defines required tokens for .github/workflows/docker-publish.yml
var ReferenceDockerPublishSpec = struct {
	Platforms        []string
	QemuAction       string
	BuildxAction     string
	LoginAction      string
	BuildPushAction  string
	DockerHubRepo    string
	GHCRRepo         string
	DockerHubSecrets []string
	GHCRSecret       string
	GhaCacheTokens   []string
}{
	Platforms:        []string{"linux/amd64", "linux/arm64"},
	QemuAction:       "docker/setup-qemu-action@v3",
	BuildxAction:     "docker/setup-buildx-action@v3",
	LoginAction:      "docker/login-action@v3",
	BuildPushAction:  "docker/build-push-action@v6",
	DockerHubRepo:    "moresonsunn/orbit",
	GHCRRepo:         "ghcr.io",
	DockerHubSecrets: []string{"DOCKERHUB_USERNAME", "DOCKERHUB_TOKEN"},
	GHCRSecret:       "GITHUB_TOKEN",
	GhaCacheTokens:   []string{"type=gha", "mode=max"},
}

// ReferenceComposeSpec defines required fields for docker-compose.prod.yml
var ReferenceComposeSpec = struct {
	ServiceName     string
	ImageName       string
	RestartPolicy   string
	HostPort        string
	ContainerPort   string
	SocketMount     string
	DataMount       string
	ProcMount       string
	SysMount        string
	SecurityOpt     string
	GroupAdd        string
	HealthcheckCmd  []string
	LogDriver       string
	LogMaxSize      string
	LogMaxFile      string
}{
	ServiceName:    "orbit",
	ImageName:      "moresonsun/orbit",
	RestartPolicy:  "unless-stopped",
	HostPort:       "8787",
	ContainerPort:  "8787",
	SocketMount:    "/var/run/docker.sock:/var/run/docker.sock",
	DataMount:      "/data",
	ProcMount:      "/proc:/host/proc:ro",
	SysMount:       "/sys:/host/sys:ro",
	SecurityOpt:    "no-new-privileges:true",
	GroupAdd:       "999",
	HealthcheckCmd: []string{"CMD", "/app/orbit", "-healthcheck"},
	LogDriver:      "json-file",
	LogMaxSize:     "20m",
	LogMaxFile:     "5",
}

// ReferenceReadmeSpec defines required sections, badges, and variables
var ReferenceReadmeSpec = struct {
	RequiredBadges []string
	Quickstarts    []string
	FeatureCount   int
	EnvironmentVars []string
}{
	RequiredBadges: []string{
		"actions/workflows/ci.yml/badge.svg",
		"img.shields.io/docker/pulls",
		"arch-amd64%20%7C%20arm64",
		"github/v/release",
		"license",
	},
	Quickstarts: []string{
		"docker run",
		"docker compose",
	},
	FeatureCount: 10,
	EnvironmentVars: []string{
		"ORBIT_LISTEN",
		"ORBIT_DATA_DIR",
		"ORBIT_DOCKER_HOST",
		"ORBIT_HOST_PROC",
		"ORBIT_HOST_SYS",
		"ORBIT_HOST_ETC",
		"ORBIT_HOST_ROOTFS",
		"ORBIT_ENABLE_ACTIONS",
		"ORBIT_ENABLE_DESTRUCTIVE",
		"ORBIT_EBPF",
		"ORBIT_GPU",
		"ORBIT_TRUST_PROXY",
		"ORBIT_SECURE_COOKIES",
		"ORBIT_UID",
		"ORBIT_GID",
		"ORBIT_KEEP_ROOT",
		"ORBIT_MAX_STATS_STREAMS",
		"ORBIT_LOG_LEVEL",
		"ORBIT_DEV",
		"ORBIT_TRUSTED_ORIGINS",
		"ORBIT_SESSION_TTL",
		"ORBIT_SESSION_IDLE",
	},
}
