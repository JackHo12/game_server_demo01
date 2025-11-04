package match

import (
	"context"
	"log"
	"time"

	"github.com/yourname/matchmaker-lite/internal/store"
	"github.com/yourname/matchmaker-lite/internal/ws"
	"github.com/yourname/matchmaker-lite/pkg/types"
)

// Simple 5v5 matcher: tries to find 10 players within widening MMR tolerance.

type Matchmaker struct {
	st  store.Store
	hub *ws.Hub
}

func NewMatchmaker(st store.Store, hub *ws.Hub) *Matchmaker { return &Matchmaker{st: st, hub: hub} }

func (m *Matchmaker) Run(ctx context.Context) {
	Ticker := time.NewTicker(500 * time.Millisecond)
	defer Ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-Ticker.C:
			m.tryMakeMatch(ctx)
		}
	}
}

func (m *Matchmaker) tryMakeMatch(ctx context.Context) {
	// Fetch candidates sorted by MMR.
	players, err := m.st.PeekQueue(ctx, 50)
	if err != nil || len(players) < 10 {
		return
	}

	// Greedy grouping: walk the list and try to form 10-player batches with growing band.
	used := map[string]bool{}
	for i := 0; i < len(players); i++ {
		if used[players[i].PlayerID] {
			continue
		}
		seed := players[i]
		band := 50.0 // starting tolerance
		var group []types.Player
		group = append(group, seed)
		for pass := 0; pass < 5 && len(group) < 10; pass++ { // widen tolerance up to 5 times
			for j := i + 1; j < len(players) && len(group) < 10; j++ {
				if used[players[j].PlayerID] {
					continue
				}
				d := abs(players[j].MMR - seed.MMR)
				if d <= band {
					group = append(group, players[j])
					used[players[j].PlayerID] = true
				}
			}
			band *= 1.5
		}
		if len(group) == 10 {
			// Remove from queue and announce match
			if err := m.st.CommitMatch(ctx, group); err != nil {
				log.Printf("commit match err: %v", err)
				return
			}
			m.hub.Broadcast(types.Event{Type: "match_found", Payload: types.Match{Players: group}})
			log.Printf("match formed: %d players", len(group))
		}
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
