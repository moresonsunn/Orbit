package api

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moby/moby/api/types/network"
	"github.com/phili/orbit/internal/alerts"
	"github.com/phili/orbit/internal/auth"
	"github.com/phili/orbit/internal/config"
	"github.com/phili/orbit/internal/docker"
	"github.com/phili/orbit/internal/gpu"
	"github.com/phili/orbit/internal/host"
	"github.com/phili/orbit/internal/metrics"
	"github.com/phili/orbit/internal/netflow"
	"github.com/phili/orbit/internal/settings"
	"github.com/phili/orbit/internal/store"
)

type Server struct {
	cfg         *config.Config
	store       *store.Store
	gw          *docker.Gateway
	inv         *docker.Inventory
	streamer    *docker.StatsStreamer
	events      *docker.EventWatcher
	hostCol     *host.Collector
	gpuMgr      *gpu.Manager
	netflowBld  *netflow.Builder
	history     *metrics.HistoryStore
	alertsEng   *alerts.Engine
	settingsMgr *settings.Manager
	authMgr     *auth.Manager
}

func NewServer(
	cfg *config.Config,
	st *store.Store,
	gw *docker.Gateway,
	inv *docker.Inventory,
	streamer *docker.StatsStreamer,
	events *docker.EventWatcher,
	hostCol *host.Collector,
	gpuMgr *gpu.Manager,
	netflowBld *netflow.Builder,
	history *metrics.HistoryStore,
	alertsEng *alerts.Engine,
	settingsMgr *settings.Manager,
	authMgr *auth.Manager,
) *Server {
	return &Server{
		cfg:         cfg,
		store:       st,
		gw:          gw,
		inv:         inv,
		streamer:    streamer,
		events:      events,
		hostCol:     hostCol,
		gpuMgr:      gpuMgr,
		netflowBld:  netflowBld,
		history:     history,
		alertsEng:   alertsEng,
		settingsMgr: settingsMgr,
		authMgr:     authMgr,
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// ---- Auth Handlers ----

func (s *Server) HandleAuthStatus(w http.ResponseWriter, r *http.Request) {
	sess, loggedIn := GetSession(r.Context())

	res := map[string]interface{}{
		"needsSetup":    s.authMgr.NeedsSetup(),
		"authenticated": loggedIn,
		"features": map[string]interface{}{
			"enableActions":     s.cfg.EnableActions,
			"enableDestructive": s.cfg.EnableDestructive,
			"ebpfMode":          s.cfg.EBPFMode,
			"gpuMode":           s.cfg.GPUMode,
		},
	}

	if loggedIn {
		res["user"] = map[string]interface{}{
			"id":       sess.UserID,
			"username": sess.Username,
			"role":     sess.Role,
		}
		res["csrfToken"] = sess.CSRFToken
	}

	writeJSON(w, http.StatusOK, res)
}

func (s *Server) HandleSetup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ungültiger Request Body")
		return
	}

	if err := s.authMgr.CompleteSetup(r.Context(), req.Token, req.Username, req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ip := r.RemoteAddr
	if s.cfg.TrustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			ip = strings.TrimSpace(parts[0])
		}
	}

	// Auto-login upon completing initial setup
	sess, token, err := s.authMgr.Login(r.Context(), req.Username, req.Password, ip, r.UserAgent())
	if err == nil {
		secure := false
		if s.cfg.SecureCookies == "true" || (s.cfg.SecureCookies == "auto" && r.TLS != nil) {
			secure = true
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "orbit_session",
			Value:    token,
			Path:     "/",
			Expires:  sess.ExpiresAt,
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteStrictMode,
		})

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Setup erfolgreich abgeschlossen.",
			"user": map[string]interface{}{
				"id":       sess.UserID,
				"username": sess.Username,
				"role":     sess.Role,
			},
			"csrfToken": sess.CSRFToken,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Setup erfolgreich abgeschlossen. Du kannst dich nun anmelden."})
}

func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ungültiger Request Body")
		return
	}

	ip := r.RemoteAddr
	if s.cfg.TrustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			ip = strings.TrimSpace(parts[0])
		}
	}

	sess, token, err := s.authMgr.Login(r.Context(), req.Username, req.Password, ip, r.UserAgent())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	secure := false
	if s.cfg.SecureCookies == "true" || (s.cfg.SecureCookies == "auto" && r.TLS != nil) {
		secure = true
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "orbit_session",
		Value:    token,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user": map[string]interface{}{
			"id":       sess.UserID,
			"username": sess.Username,
			"role":     sess.Role,
		},
		"csrfToken": sess.CSRFToken,
	})
}

func (s *Server) HandleLogout(w http.ResponseWriter, r *http.Request) {
	var token string
	if cookie, err := r.Cookie("orbit_session"); err == nil {
		token = cookie.Value
	}
	s.authMgr.Logout(r.Context(), token)

	http.SetCookie(w, &http.Cookie{
		Name:     "orbit_session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "Erfolgreich abgemeldet."})
}

// ---- System & Host Handlers ----

func (s *Server) HandleSystemInfo(w http.ResponseWriter, r *http.Request) {
	dockerInfo, _ := s.gw.Info(r.Context())
	dockerConnected := s.gw.Connected()
	lastDockerErr := s.gw.LastError()

	hostMetrics := s.hostCol.Collect(r.Context())

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"dockerConnected": dockerConnected,
		"dockerError":     lastDockerErr,
		"docker":          dockerInfo,
		"host":            hostMetrics.System,
		"inventoryCounts": s.inv.Counts(),
	})
}

func (s *Server) HandleSystemMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := s.hostCol.Collect(r.Context())
	writeJSON(w, http.StatusOK, metrics)
}

// ---- Containers Handlers ----

type EnrichedContainer struct {
	docker.Container
	Stats *docker.ContainerStats `json:"stats,omitempty"`
}

func (s *Server) HandleListContainers(w http.ResponseWriter, r *http.Request) {
	containers := s.inv.Containers()
	enriched := make([]EnrichedContainer, len(containers))

	for i, c := range containers {
		enriched[i] = EnrichedContainer{Container: c}
		if st, ok := s.streamer.GetLatest(c.ID); ok {
			enriched[i].Stats = &st
		}
	}

	writeJSON(w, http.StatusOK, enriched)
}

func (s *Server) HandleGetContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, found := s.inv.Get(id)
	if !found {
		writeError(w, http.StatusNotFound, "Container nicht gefunden")
		return
	}

	resp := EnrichedContainer{Container: c}
	if st, ok := s.streamer.GetLatest(c.ID); ok {
		resp.Stats = &st
	}

	// Environment variables: only display to administrators as required by prompt Section 6
	sess, ok := GetSession(r.Context())
	isAdmin := ok && sess != nil && sess.Role == auth.RoleAdmin

	in, rawInspect, err := s.gw.InspectContainer(r.Context(), c.ID)
	if err == nil {
		inspectData := map[string]interface{}{
			"container": resp,
			"raw":       json.RawMessage(rawInspect),
		}
		if !isAdmin && in.Config != nil {
			// Mask env variables for non-admin users
			maskedInspect := in
			maskedInspect.Config.Env = []string{"[Aus Sicherheitsgründen nur für Administratoren sichtbar]"}
			inspectData["inspect"] = maskedInspect
		} else {
			inspectData["inspect"] = in
		}
		writeJSON(w, http.StatusOK, inspectData)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) HandleContainerStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	st, ok := s.streamer.GetLatest(id)
	if !ok {
		writeError(w, http.StatusNotFound, "Keine Live-Statistiken für diesen Container vorhanden")
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) HandleContainerStatsHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	hist := s.streamer.GetRecentHistory(id)
	if hist == nil {
		hist = []docker.ContainerStats{}
	}
	writeJSON(w, http.StatusOK, hist)
}

func (s *Server) HandleContainerLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "200"
	}

	since := r.URL.Query().Get("since")
	timestamps := r.URL.Query().Get("timestamps") == "true"

	reader, err := s.gw.Logs(r.Context(), id, docker.LogOptions{
		Tail:       tail,
		Since:      since,
		Timestamps: timestamps,
		Follow:     false,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Logs konnten nicht abgerufen werden: "+err.Error())
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

func (s *Server) HandleContainerTop(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	top, err := s.gw.Top(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Prozesse konnten nicht ermittelt werden: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, top)
}

// ---- Docker Resources Handlers ----

// NetworkContainerEndpoint represents a container connected to a network with status details.
type NetworkContainerEndpoint struct {
	Name        string `json:"Name"`
	EndpointID  string `json:"EndpointID"`
	MacAddress  string `json:"MacAddress"`
	IPv4Address string `json:"IPv4Address"`
	IPv6Address string `json:"IPv6Address"`
	State       string `json:"State,omitempty"`
}

// SubnetDetails contains IPAM capacity and allocation metrics.
type SubnetDetails struct {
	Subnet             string  `json:"Subnet"`
	Gateway            string  `json:"Gateway"`
	NetworkIP          string  `json:"NetworkIP"`
	BroadcastIP        string  `json:"BroadcastIP"`
	UsableRange        string  `json:"UsableRange"`
	TotalIPs           int     `json:"TotalIPs"`
	UsedIPs            int     `json:"UsedIPs"`
	FreeIPs            int     `json:"FreeIPs"`
	UtilizationPercent float64 `json:"UtilizationPercent"`
}

// EnrichedDockerNetwork delivers network metadata, IPAM calculations, and connected containers.
// It embeds network.Summary for full backward compatibility with standard Docker network inspects.
type EnrichedDockerNetwork struct {
	network.Summary
	SubnetDetails *SubnetDetails                      `json:"SubnetDetails,omitempty"`
	Containers    map[string]NetworkContainerEndpoint `json:"Containers"`
	Services      map[string]network.ServiceInfo      `json:"Services,omitempty"`
	Status        *network.Status                     `json:"Status,omitempty"`
}

func calculateSubnetDetails(subnetStr, gatewayStr string, containers map[string]NetworkContainerEndpoint, status *network.Status) *SubnetDetails {
	if subnetStr == "" {
		return nil
	}

	_, ipnet, err := net.ParseCIDR(subnetStr)
	if err != nil {
		return nil
	}

	ip4 := ipnet.IP.To4()
	if ip4 == nil {
		return nil
	}

	ones, bits := ipnet.Mask.Size()
	if bits != 32 {
		return nil
	}
	hostBits := 32 - ones

	networkIP := ip4.String()

	broadcast := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		broadcast[i] = ip4[i] | ^ipnet.Mask[i]
	}
	broadcastIP := broadcast.String()

	netUint := binary.BigEndian.Uint32(ip4)
	bcastUint := binary.BigEndian.Uint32(broadcast)

	var usableRange string
	if hostBits >= 2 {
		firstUsable := make(net.IP, 4)
		lastUsable := make(net.IP, 4)
		binary.BigEndian.PutUint32(firstUsable, netUint+1)
		binary.BigEndian.PutUint32(lastUsable, bcastUint-1)
		usableRange = fmt.Sprintf("%s - %s", firstUsable.String(), lastUsable.String())
	} else if hostBits == 1 {
		usableRange = fmt.Sprintf("%s - %s", networkIP, broadcastIP)
	} else {
		usableRange = networkIP
	}

	totalIPs := 1 << hostBits

	usedIPs := 0
	freeIPs := 0

	// Check if Docker status has runtime IPAM allocations
	if status != nil && status.IPAM.Subnets != nil {
		for pfx, subStatus := range status.IPAM.Subnets {
			if pfx.String() == subnetStr {
				if subStatus.IPsInUse > 0 || subStatus.DynamicIPsAvailable > 0 {
					usedIPs = int(subStatus.IPsInUse)
					freeIPs = int(subStatus.DynamicIPsAvailable)
					if usedIPs+freeIPs > totalIPs {
						totalIPs = usedIPs + freeIPs
					}
					break
				}
			}
		}
	}

	// Fallback calculation from connected containers and gateway
	if usedIPs == 0 {
		count := 0
		for _, c := range containers {
			if c.IPv4Address != "" {
				cIPStr := strings.Split(c.IPv4Address, "/")[0]
				if parsed := net.ParseIP(cIPStr); parsed != nil && ipnet.Contains(parsed) {
					count++
				}
			}
		}
		if gatewayStr != "" {
			if gwIP := net.ParseIP(gatewayStr); gwIP != nil && ipnet.Contains(gwIP) {
				count++
			}
		}
		usedIPs = count
		freeIPs = totalIPs - usedIPs
		if freeIPs < 0 {
			freeIPs = 0
		}
	}

	utilPct := 0.0
	if totalIPs > 0 {
		utilPct = math.Round((float64(usedIPs)/float64(totalIPs)*100.0)*100) / 100
	}

	return &SubnetDetails{
		Subnet:             subnetStr,
		Gateway:            gatewayStr,
		NetworkIP:          networkIP,
		BroadcastIP:        broadcastIP,
		UsableRange:        usableRange,
		TotalIPs:           totalIPs,
		UsedIPs:            usedIPs,
		FreeIPs:            freeIPs,
		UtilizationPercent: utilPct,
	}
}

func (s *Server) HandleListNetworks(w http.ResponseWriter, r *http.Request) {
	nets, err := s.inv.Networks(r.Context(), 10*time.Second)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Netzwerke konnten nicht geladen werden: "+err.Error())
		return
	}

	containers := s.inv.Containers()

	var enriched []EnrichedDockerNetwork
	for _, n := range nets {
		netContainers := make(map[string]NetworkContainerEndpoint)

		for _, c := range containers {
			for _, cn := range c.Networks {
				matches := cn.NetworkID == n.ID || cn.Name == n.Name
				if !matches && len(n.ID) >= 12 && len(cn.NetworkID) >= 12 {
					matches = strings.HasPrefix(cn.NetworkID, n.ID[:12]) || strings.HasPrefix(n.ID, cn.NetworkID[:12])
				}
				if matches {
					ipv4 := cn.IPAddress
					if ipv4 != "" && cn.PrefixLen > 0 {
						ipv4 = fmt.Sprintf("%s/%d", cn.IPAddress, cn.PrefixLen)
					}
					netContainers[c.ID] = NetworkContainerEndpoint{
						Name:        c.Name,
						EndpointID:  cn.EndpointID,
						MacAddress:  cn.MacAddress,
						IPv4Address: ipv4,
						IPv6Address: cn.IPv6,
						State:       c.State,
					}
				}
			}
		}

		var subnetDetails *SubnetDetails
		for _, ipamCfg := range n.IPAM.Config {
			if ipamCfg.Subnet.IsValid() {
				subnetStr := ipamCfg.Subnet.String()
				gatewayStr := ""
				if ipamCfg.Gateway.IsValid() {
					gatewayStr = ipamCfg.Gateway.String()
				}
				subnetDetails = calculateSubnetDetails(subnetStr, gatewayStr, netContainers, nil)
				break
			}
		}

		enriched = append(enriched, EnrichedDockerNetwork{
			Summary:       n,
			SubnetDetails: subnetDetails,
			Containers:    netContainers,
		})
	}

	writeJSON(w, http.StatusOK, enriched)
}

func (s *Server) HandleGetNetwork(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	netInspect, err := s.gw.InspectNetwork(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Netzwerk nicht gefunden: "+err.Error())
		return
	}

	containers := s.inv.Containers()
	containerMap := make(map[string]docker.Container, len(containers))
	for _, c := range containers {
		containerMap[c.ID] = c
	}

	netContainers := make(map[string]NetworkContainerEndpoint)

	// Populate from inspected containers
	for cid, ep := range netInspect.Containers {
		state := ""
		if c, ok := containerMap[cid]; ok {
			state = c.State
		} else {
			for _, c := range containers {
				if c.ID == cid || c.ShortID == cid || strings.HasPrefix(cid, c.ShortID) || c.Name == ep.Name {
					state = c.State
					break
				}
			}
		}
		ipv4Str := ""
		if ep.IPv4Address.IsValid() {
			ipv4Str = ep.IPv4Address.String()
		}
		ipv6Str := ""
		if ep.IPv6Address.IsValid() {
			ipv6Str = ep.IPv6Address.String()
		}
		macStr := ""
		if len(ep.MacAddress) > 0 {
			macStr = ep.MacAddress.String()
		}

		netContainers[cid] = NetworkContainerEndpoint{
			Name:        ep.Name,
			EndpointID:  ep.EndpointID,
			MacAddress:  macStr,
			IPv4Address: ipv4Str,
			IPv6Address: ipv6Str,
			State:       state,
		}
	}

	// Also ensure any containers in inventory attached to this network are included
	for _, c := range containers {
		for _, cn := range c.Networks {
			matches := cn.NetworkID == netInspect.ID || cn.Name == netInspect.Name
			if !matches && len(netInspect.ID) >= 12 && len(cn.NetworkID) >= 12 {
				matches = strings.HasPrefix(cn.NetworkID, netInspect.ID[:12]) || strings.HasPrefix(netInspect.ID, cn.NetworkID[:12])
			}
			if matches {
				if _, exists := netContainers[c.ID]; !exists {
					ipv4 := cn.IPAddress
					if ipv4 != "" && cn.PrefixLen > 0 {
						ipv4 = fmt.Sprintf("%s/%d", cn.IPAddress, cn.PrefixLen)
					}
					netContainers[c.ID] = NetworkContainerEndpoint{
						Name:        c.Name,
						EndpointID:  cn.EndpointID,
						MacAddress:  cn.MacAddress,
						IPv4Address: ipv4,
						IPv6Address: cn.IPv6,
						State:       c.State,
					}
				}
			}
		}
	}

	var subnetDetails *SubnetDetails
	for _, ipamCfg := range netInspect.IPAM.Config {
		if ipamCfg.Subnet.IsValid() {
			subnetStr := ipamCfg.Subnet.String()
			gatewayStr := ""
			if ipamCfg.Gateway.IsValid() {
				gatewayStr = ipamCfg.Gateway.String()
			}
			subnetDetails = calculateSubnetDetails(subnetStr, gatewayStr, netContainers, netInspect.Status)
			break
		}
	}

	enriched := EnrichedDockerNetwork{
		Summary:       network.Summary{Network: netInspect.Network},
		SubnetDetails: subnetDetails,
		Containers:    netContainers,
		Services:      netInspect.Services,
		Status:        netInspect.Status,
	}

	writeJSON(w, http.StatusOK, enriched)
}

func (s *Server) HandleListImages(w http.ResponseWriter, r *http.Request) {
	imgs, err := s.inv.Images(r.Context(), 10*time.Second)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Images konnten nicht geladen werden: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, imgs)
}

func (s *Server) HandleListVolumes(w http.ResponseWriter, r *http.Request) {
	vols, err := s.inv.Volumes(r.Context(), 10*time.Second)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Volumes konnten nicht geladen werden: "+err.Error())
		return
	}

	// Try to get volume sizes
	sizes, _ := s.gw.VolumeSizes(r.Context())

	type EnrichedVolume struct {
		docker.Mount
		Volume    interface{} `json:"volume"`
		SizeBytes int64       `json:"sizeBytes,omitempty"`
	}

	var enriched []interface{}
	for _, v := range vols {
		item := map[string]interface{}{
			"name":       v.Name,
			"driver":     v.Driver,
			"mountpoint": v.Mountpoint,
			"createdAt":  v.CreatedAt,
			"labels":     v.Labels,
			"scope":      v.Scope,
		}
		if sizes != nil {
			if sz, ok := sizes[v.Name]; ok {
				item["sizeBytes"] = sz
			}
		}
		enriched = append(enriched, item)
	}

	writeJSON(w, http.StatusOK, enriched)
}

// ---- Network Flow, GPU, Events, Alerts Handlers ----

func (s *Server) HandleNetflow(w http.ResponseWriter, r *http.Request) {
	snapshot := s.netflowBld.BuildSnapshot(r.Context())
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) HandleGPU(w http.ResponseWriter, r *http.Request) {
	telemetry := s.gpuMgr.Collect(r.Context())
	writeJSON(w, http.StatusOK, telemetry)
}

func (s *Server) HandleEvents(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	evs, err := s.events.RecentEvents(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Events konnten nicht abgerufen werden: "+err.Error())
		return
	}
	if evs == nil {
		evs = []docker.DockerEvent{}
	}
	writeJSON(w, http.StatusOK, evs)
}

func (s *Server) HandleAlerts(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	allAlerts, err := s.alertsEng.RecentAlerts(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Alerts konnten nicht abgerufen werden: "+err.Error())
		return
	}
	if allAlerts == nil {
		allAlerts = []alerts.Alert{}
	}
	writeJSON(w, http.StatusOK, allAlerts)
}

func (s *Server) HandleAuditLog(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	rows, err := s.store.DB.QueryContext(r.Context(),
		`SELECT id, ts, COALESCE(user_id, 0), COALESCE(username, ''), action, COALESCE(target, ''), result, COALESCE(ip, ''), COALESCE(details, '')
		 FROM audit_log ORDER BY ts DESC, id DESC LIMIT ?`, limit,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Audit-Log konnte nicht abgerufen werden: "+err.Error())
		return
	}
	defer rows.Close()

	type AuditEntry struct {
		ID        int64  `json:"id"`
		Timestamp int64  `json:"timestamp"`
		UserID    int64  `json:"userId"`
		Username  string `json:"username"`
		Action    string `json:"action"`
		Target    string `json:"target"`
		Result    string `json:"result"`
		IP        string `json:"ip"`
		Details   string `json:"details"`
	}

	entries := make([]AuditEntry, 0)
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.UserID, &e.Username, &e.Action, &e.Target, &e.Result, &e.IP, &e.Details); err == nil {
			entries = append(entries, e)
		}
	}

	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) HandleAckAlert(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ungültige Alert ID")
		return
	}

	if err := s.alertsEng.Acknowledge(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Alert konnte nicht bestätigt werden: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Alert bestätigt."})
}

func (s *Server) HandleGetSettings(w http.ResponseWriter, r *http.Request) {
	cur := s.settingsMgr.Get()
	writeJSON(w, http.StatusOK, cur)
}

func (s *Server) HandleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req settings.Settings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ungültige Einstellungen: "+err.Error())
		return
	}

	if err := s.settingsMgr.Update(r.Context(), req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, req)
}

func (s *Server) HandleMetricsHistory(w http.ResponseWriter, r *http.Request) {
	series := r.URL.Query().Get("series")
	if series == "" {
		writeError(w, http.StatusBadRequest, "Parameter 'series' erforderlich")
		return
	}

	now := time.Now().Unix()
	from := now - 3600 // default 1 hour
	to := now

	if f := r.URL.Query().Get("from"); f != "" {
		if val, err := strconv.ParseInt(f, 10, 64); err == nil {
			from = val
		}
	}
	if t := r.URL.Query().Get("to"); t != "" {
		if val, err := strconv.ParseInt(t, 10, 64); err == nil {
			to = val
		}
	}

	resolution := r.URL.Query().Get("resolution")
	hist, err := s.history.Query(r.Context(), series, from, to, resolution)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Historie konnte nicht geladen werden: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, hist)
}
