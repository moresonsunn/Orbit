package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/phili/orbit/internal/realtime"
)

func SetupRouter(server *Server, hub *realtime.Hub, spaFS fs.FS) http.Handler {
	r := chi.NewRouter()

	mw := NewMiddleware(server.cfg, server.authMgr)

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(mw.SecurityHeaders)
	r.Use(mw.CORS)
	r.Use(mw.Authenticate)

	// API routes
	r.Route("/api/v1", func(api chi.Router) {
		// Health check
		api.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})

		// WebSocket endpoint (handles its own auth and origin checks)
		api.Get("/ws", hub.HandleWebSocket)

		// Public Auth routes
		api.Get("/auth/status", server.HandleAuthStatus)
		api.Post("/auth/setup", server.HandleSetup)
		api.Post("/auth/login", server.HandleLogin)
		api.Post("/auth/logout", server.HandleLogout)

		// Authenticated Routes
		api.Group(func(authGroup chi.Router) {
			authGroup.Use(mw.RequireAuth)
			authGroup.Use(mw.VerifyCSRF)

			// System & Monitoring
			authGroup.Get("/system", server.HandleSystemInfo)
			authGroup.Get("/system/metrics", server.HandleSystemMetrics)
			authGroup.Get("/metrics/history", server.HandleMetricsHistory)

			// Containers Read
			authGroup.Get("/containers", server.HandleListContainers)
			authGroup.Get("/containers/{id}", server.HandleGetContainer)
			authGroup.Get("/containers/{id}/stats", server.HandleContainerStats)
			authGroup.Get("/containers/{id}/stats/history", server.HandleContainerStatsHistory)
			authGroup.Get("/containers/{id}/logs", server.HandleContainerLogs)
			authGroup.Get("/containers/{id}/top", server.HandleContainerTop)

			// Docker Resources Read
			authGroup.Get("/networks", server.HandleListNetworks)
			authGroup.Get("/networks/{id}", server.HandleGetNetwork)
			authGroup.Get("/images", server.HandleListImages)
			authGroup.Get("/volumes", server.HandleListVolumes)

			// Network Flow & GPU
			authGroup.Get("/netflow", server.HandleNetflow)
			authGroup.Get("/gpu", server.HandleGPU)

			// Events & Alerts & Audit
			authGroup.Get("/events", server.HandleEvents)
			authGroup.Get("/alerts", server.HandleAlerts)
			authGroup.Post("/alerts/{id}/ack", server.HandleAckAlert)
			authGroup.Get("/audit", server.HandleAuditLog)

			// Settings
			authGroup.Get("/settings", server.HandleGetSettings)

			// Admin-Only Routes
			authGroup.Group(func(admin chi.Router) {
				admin.Use(mw.RequireAdmin)

				admin.Put("/settings", server.HandleUpdateSettings)

				// Container Management Actions
				admin.Post("/containers/{id}/start", server.HandleContainerStart)
				admin.Post("/containers/{id}/stop", server.HandleContainerStop)
				admin.Post("/containers/{id}/restart", server.HandleContainerRestart)
				admin.Post("/containers/{id}/pause", server.HandleContainerPause)
				admin.Post("/containers/{id}/unpause", server.HandleContainerUnpause)
				admin.Delete("/containers/{id}", server.HandleContainerRemove)

				// Bulk Actions
				admin.Post("/containers/bulk/start", server.HandleBulkStart)
				admin.Post("/containers/bulk/stop", server.HandleBulkStop)
				admin.Post("/containers/bulk/restart", server.HandleBulkRestart)

				// Resources Management
				admin.Delete("/volumes/{name}", server.HandleVolumeRemove)
				admin.Post("/volumes/prune", server.HandleVolumePrune)
				admin.Delete("/images/{id}", server.HandleImageRemove)
				admin.Post("/images/prune", server.HandleImagePrune)
				admin.Delete("/networks/{id}", server.HandleNetworkRemove)
			})
		})
	})

	// Serve Frontend SPA static files
	if spaFS != nil {
		fileServer := http.FileServer(http.FS(spaFS))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				fileServer.ServeHTTP(w, r)
				return
			}

			// Check if file physically exists in spaFS
			if f, err := spaFS.Open(path); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}

			// Fallback to index.html for SPA client-side routes
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
		})
	}

	return r
}
