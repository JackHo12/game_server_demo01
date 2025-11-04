package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/yourname/matchmaker-lite/pkg/types"
)

type Hub struct {
	clients map[*websocket.Conn]bool
	broadcast chan types.Event
	upgrade websocket.Upgrader
}

func NewHub() *Hub {
	return &Hub{
		clients:   map[*websocket.Conn]bool{},
		broadcast: make(chan types.Event, 64),
		upgrade: websocket.Upgrader{ CheckOrigin: func(*http.Request) bool { return true } },
	}
}

func (h *Hub) Run() {
	for ev := range h.broadcast {
		for c := range h.clients {
			if err := c.WriteJSON(ev); err != nil { log.Printf("ws write err: %v", err); c.Close(); delete(h.clients, c) }
		}
	}
}

func (h *Hub) Broadcast(ev types.Event) { h.broadcast <- ev }

func ServeWS(h *Hub, w http.ResponseWriter, r *http.Request) {
	c, err := h.upgrade.Upgrade(w, r, nil)
	if err != nil { http.Error(w, err.Error(), 400); return }
	h.clients[c] = true
}