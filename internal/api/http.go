package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yourname/matchmaker-lite/internal/match"
	"github.com/yourname/matchmaker-lite/internal/store"
	"github.com/yourname/matchmaker-lite/internal/ws"
	"github.com/yourname/matchmaker-lite/pkg/types"
)

type router struct {
	st store.Store
	h  *ws.Hub
	mm *match.Matchmaker
}

func NewRouter(st store.Store, h *ws.Hub, mm *match.Matchmaker) http.Handler {
	r := &router{st: st, h: h, mm: mm}

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)

	mux.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.Handle("/metrics", promhttp.Handler())

	mux.Post("/join", r.handleJoin)
	mux.Post("/leave", r.handleLeave)
	mux.Get("/ws", r.handleWS)

	return mux
}

func (r *router) handleJoin(w http.ResponseWriter, req *http.Request) {
	var p types.JoinRequest
	if err := json.NewDecoder(req.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if p.PlayerID == "" {
		http.Error(w, "missing player_id", http.StatusBadRequest)
		return
	}
	if p.MMR <= 0 {
		http.Error(w, "invalid mmr", http.StatusBadRequest)
		return
	}

	jid, err := r.st.Enqueue(req.Context(), p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"enqueued": true, "job_id": jid, "at": time.Now()})
}

func (r *router) handleLeave(w http.ResponseWriter, req *http.Request) {
	var p struct {
		PlayerID string `json:"player_id"`
	}
	if err := json.NewDecoder(req.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if p.PlayerID == "" {
		http.Error(w, "missing player_id", http.StatusBadRequest)
		return
	}
	if err := r.st.Dequeue(req.Context(), p.PlayerID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"removed": true})
}

func (r *router) handleWS(w http.ResponseWriter, req *http.Request) {
	ws.ServeWS(r.h, w, req)
}
