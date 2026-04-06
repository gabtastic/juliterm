package server

import (
	"encoding/json"
	"log"
	"net/http"
)

// handleSessions returns the current tmux session tree as JSON.
// GET /api/sessions
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	treeJSON, err := s.tmuxMgr.Tree().ToJSON()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to serialize session tree"})
		return
	}

	log.Printf("api: GET /api/sessions returning %d bytes (%d sessions)", len(treeJSON), len(s.tmuxMgr.Tree().GetSessions()))

	w.Header().Set("Content-Type", "application/json")
	w.Write(treeJSON)
}
