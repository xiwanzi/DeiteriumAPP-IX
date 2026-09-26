package httpapi

import "net/http"

func (s *Server) chatPresence(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.socialUserV2(w, r); !ok {
		return
	}
	nodes := []string{}
	for _, node := range s.Config.Nodes {
		if node.Chat {
			nodes = append(nodes, node.ID)
		}
	}
	snapshot, available := s.Core.Presence(nodes)
	players, err := s.Store.ChatOnlinePlayers(r.Context(), snapshot)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	// Core snapshots do not carry a capture timestamp; do not invent updatedAt.
	success(w, r, map[string]any{"available": available, "onlineCount": len(players), "players": players, "updatedAt": nil})
}
