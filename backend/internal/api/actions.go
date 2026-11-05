package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ---- Container Actions ----

func (s *Server) checkActionsAllowed(w http.ResponseWriter) bool {
	if !s.cfg.EnableActions {
		writeError(w, http.StatusForbidden, "Container-Verwaltungsaktionen sind deaktiviert (ORBIT_ENABLE_ACTIONS=false). Orbit läuft im reinen Monitoring-Modus.")
		return false
	}
	return true
}

func (s *Server) checkDestructiveAllowed(w http.ResponseWriter) bool {
	if !s.cfg.EnableActions {
		writeError(w, http.StatusForbidden, "Verwaltungsaktionen sind deaktiviert (ORBIT_ENABLE_ACTIONS=false).")
		return false
	}
	if !s.cfg.EnableDestructive {
		writeError(w, http.StatusForbidden, "Destruktive Aktionen (Löschen/Prune) sind deaktiviert (ORBIT_ENABLE_DESTRUCTIVE=false).")
		return false
	}
	return true
}

func (s *Server) HandleContainerStart(w http.ResponseWriter, r *http.Request) {
	if !s.checkActionsAllowed(w) {
		return
	}
	id := chi.URLParam(r, "id")
	sess, _ := GetSession(r.Context())

	if err := s.gw.Start(r.Context(), id); err != nil {
		s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.start", id, "failed", r.RemoteAddr, err.Error())
		writeError(w, http.StatusInternalServerError, "Container konnte nicht gestartet werden: "+err.Error())
		return
	}

	s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.start", id, "success", r.RemoteAddr, "")
	s.inv.Refresh(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Container gestartet."})
}

func (s *Server) HandleContainerStop(w http.ResponseWriter, r *http.Request) {
	if !s.checkActionsAllowed(w) {
		return
	}
	id := chi.URLParam(r, "id")
	sess, _ := GetSession(r.Context())

	if err := s.gw.Stop(r.Context(), id); err != nil {
		s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.stop", id, "failed", r.RemoteAddr, err.Error())
		writeError(w, http.StatusInternalServerError, "Container konnte nicht gestoppt werden: "+err.Error())
		return
	}

	s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.stop", id, "success", r.RemoteAddr, "")
	s.inv.Refresh(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Container gestoppt."})
}

func (s *Server) HandleContainerRestart(w http.ResponseWriter, r *http.Request) {
	if !s.checkActionsAllowed(w) {
		return
	}
	id := chi.URLParam(r, "id")
	sess, _ := GetSession(r.Context())

	if err := s.gw.Restart(r.Context(), id); err != nil {
		s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.restart", id, "failed", r.RemoteAddr, err.Error())
		writeError(w, http.StatusInternalServerError, "Container konnte nicht neugestartet werden: "+err.Error())
		return
	}

	s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.restart", id, "success", r.RemoteAddr, "")
	s.inv.Refresh(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Container neugestartet."})
}

func (s *Server) HandleContainerPause(w http.ResponseWriter, r *http.Request) {
	if !s.checkActionsAllowed(w) {
		return
	}
	id := chi.URLParam(r, "id")
	sess, _ := GetSession(r.Context())

	if err := s.gw.Pause(r.Context(), id); err != nil {
		s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.pause", id, "failed", r.RemoteAddr, err.Error())
		writeError(w, http.StatusInternalServerError, "Container konnte nicht pausiert werden: "+err.Error())
		return
	}

	s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.pause", id, "success", r.RemoteAddr, "")
	s.inv.Refresh(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Container pausiert."})
}

func (s *Server) HandleContainerUnpause(w http.ResponseWriter, r *http.Request) {
	if !s.checkActionsAllowed(w) {
		return
	}
	id := chi.URLParam(r, "id")
	sess, _ := GetSession(r.Context())

	if err := s.gw.Unpause(r.Context(), id); err != nil {
		s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.unpause", id, "failed", r.RemoteAddr, err.Error())
		writeError(w, http.StatusInternalServerError, "Container konnte nicht fortgesetzt werden: "+err.Error())
		return
	}

	s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.unpause", id, "success", r.RemoteAddr, "")
	s.inv.Refresh(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Container fortgesetzt."})
}

func (s *Server) HandleContainerRemove(w http.ResponseWriter, r *http.Request) {
	if !s.checkDestructiveAllowed(w) {
		return
	}
	id := chi.URLParam(r, "id")
	force := r.URL.Query().Get("force") == "true"
	sess, _ := GetSession(r.Context())

	if err := s.gw.RemoveContainer(r.Context(), id, force); err != nil {
		s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.remove", id, "failed", r.RemoteAddr, err.Error())
		writeError(w, http.StatusInternalServerError, "Container konnte nicht entfernt werden: "+err.Error())
		return
	}

	s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.remove", id, "success", r.RemoteAddr, "")
	s.inv.Refresh(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Container erfolgreich entfernt."})
}

// ---- Bulk Actions ----

type BulkActionRequest struct {
	IDs []string `json:"ids"`
}

type BulkActionResult struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (s *Server) HandleBulkStart(w http.ResponseWriter, r *http.Request) {
	if !s.checkActionsAllowed(w) {
		return
	}
	var req BulkActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ungültiger Request Body")
		return
	}

	sess, _ := GetSession(r.Context())
	var results []BulkActionResult

	for _, id := range req.IDs {
		err := s.gw.Start(r.Context(), id)
		res := BulkActionResult{ID: id, Success: err == nil}
		if err != nil {
			res.Error = err.Error()
			s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.start", id, "failed", r.RemoteAddr, err.Error())
		} else {
			s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.start", id, "success", r.RemoteAddr, "")
			s.inv.Refresh(r.Context(), id)
		}
		results = append(results, res)
	}

	writeJSON(w, http.StatusOK, results)
}

func (s *Server) HandleBulkStop(w http.ResponseWriter, r *http.Request) {
	if !s.checkActionsAllowed(w) {
		return
	}
	var req BulkActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ungültiger Request Body")
		return
	}

	sess, _ := GetSession(r.Context())
	var results []BulkActionResult

	for _, id := range req.IDs {
		err := s.gw.Stop(r.Context(), id)
		res := BulkActionResult{ID: id, Success: err == nil}
		if err != nil {
			res.Error = err.Error()
			s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.stop", id, "failed", r.RemoteAddr, err.Error())
		} else {
			s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.stop", id, "success", r.RemoteAddr, "")
			s.inv.Refresh(r.Context(), id)
		}
		results = append(results, res)
	}

	writeJSON(w, http.StatusOK, results)
}

func (s *Server) HandleBulkRestart(w http.ResponseWriter, r *http.Request) {
	if !s.checkActionsAllowed(w) {
		return
	}
	var req BulkActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ungültiger Request Body")
		return
	}

	sess, _ := GetSession(r.Context())
	var results []BulkActionResult

	for _, id := range req.IDs {
		err := s.gw.Restart(r.Context(), id)
		res := BulkActionResult{ID: id, Success: err == nil}
		if err != nil {
			res.Error = err.Error()
			s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.restart", id, "failed", r.RemoteAddr, err.Error())
		} else {
			s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "container.restart", id, "success", r.RemoteAddr, "")
			s.inv.Refresh(r.Context(), id)
		}
		results = append(results, res)
	}

	writeJSON(w, http.StatusOK, results)
}

// ---- Volume & Image Destructive Actions ----

func (s *Server) HandleVolumeRemove(w http.ResponseWriter, r *http.Request) {
	if !s.checkDestructiveAllowed(w) {
		return
	}
	name := chi.URLParam(r, "name")

	// Safety check: is this volume used by any existing container?
	for _, c := range s.inv.Containers() {
		for _, m := range c.Mounts {
			if m.Name == name {
				writeError(w, http.StatusConflict, fmt.Sprintf("Volume kann nicht gelöscht werden: Es wird vom Container '%s' (%s) verwendet.", c.Name, c.State))
				return
			}
		}
	}

	sess, _ := GetSession(r.Context())
	if err := s.gw.RemoveVolume(r.Context(), name); err != nil {
		s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "volume.remove", name, "failed", r.RemoteAddr, err.Error())
		writeError(w, http.StatusInternalServerError, "Volume konnte nicht gelöscht werden: "+err.Error())
		return
	}

	s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "volume.remove", name, "success", r.RemoteAddr, "")
	s.inv.RefreshResources(r.Context(), "volumes")
	writeJSON(w, http.StatusOK, map[string]string{"message": "Volume gelöscht."})
}

func (s *Server) HandleVolumePrune(w http.ResponseWriter, r *http.Request) {
	if !s.checkDestructiveAllowed(w) {
		return
	}
	all := r.URL.Query().Get("all") == "true"
	sess, _ := GetSession(r.Context())

	report, err := s.gw.PruneVolumes(r.Context(), all)
	if err != nil {
		s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "volume.prune", "", "failed", r.RemoteAddr, err.Error())
		writeError(w, http.StatusInternalServerError, "Volumes Prune fehlgeschlagen: "+err.Error())
		return
	}

	s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "volume.prune", "", "success", r.RemoteAddr, fmt.Sprintf("deleted: %d", len(report.VolumesDeleted)))
	s.inv.RefreshResources(r.Context(), "volumes")
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) HandleImageRemove(w http.ResponseWriter, r *http.Request) {
	if !s.checkDestructiveAllowed(w) {
		return
	}
	id := chi.URLParam(r, "id")
	sess, _ := GetSession(r.Context())

	items, err := s.gw.RemoveImage(r.Context(), id)
	if err != nil {
		s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "image.remove", id, "failed", r.RemoteAddr, err.Error())
		writeError(w, http.StatusInternalServerError, "Image konnte nicht entfernt werden: "+err.Error())
		return
	}

	s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "image.remove", id, "success", r.RemoteAddr, "")
	s.inv.RefreshResources(r.Context(), "images")
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) HandleImagePrune(w http.ResponseWriter, r *http.Request) {
	if !s.checkDestructiveAllowed(w) {
		return
	}
	all := r.URL.Query().Get("all") == "true"
	sess, _ := GetSession(r.Context())

	report, err := s.gw.PruneImages(r.Context(), all)
	if err != nil {
		s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "image.prune", "", "failed", r.RemoteAddr, err.Error())
		writeError(w, http.StatusInternalServerError, "Image Prune fehlgeschlagen: "+err.Error())
		return
	}

	s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "image.prune", "", "success", r.RemoteAddr, fmt.Sprintf("deleted: %d", len(report.ImagesDeleted)))
	s.inv.RefreshResources(r.Context(), "images")
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) HandleNetworkRemove(w http.ResponseWriter, r *http.Request) {
	if !s.checkDestructiveAllowed(w) {
		return
	}
	id := chi.URLParam(r, "id")
	sess, _ := GetSession(r.Context())

	if err := s.gw.RemoveNetwork(r.Context(), id); err != nil {
		s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "network.remove", id, "failed", r.RemoteAddr, err.Error())
		writeError(w, http.StatusInternalServerError, "Netzwerk konnte nicht entfernt werden: "+err.Error())
		return
	}

	s.authMgr.RecordAudit(r.Context(), sess.UserID, sess.Username, "network.remove", id, "success", r.RemoteAddr, "")
	s.inv.RefreshResources(r.Context(), "networks")
	writeJSON(w, http.StatusOK, map[string]string{"message": "Netzwerk entfernt."})
}

