package game

func (e *Engine) communicate() {
	next := map[string][]Message{}
	for i := range e.players {
		p := &e.players[i]
		if !p.Alive || p.Brain <= 0 {
			continue
		}
		msgs := e.agents[p.ID].Communicate(e.observe(p.ID), e.Memory(p.ID), e.rng)
		for j, m := range msgs {
			if j >= 8 || p.Brain < e.config.MessageCost {
				break
			}
			if len(m.Type) > 64 || len(m.Content) > 2048 || m.Type == "" {
				continue
			}
			m.From = p.ID
			m.Round = e.round
			m.Institution = p.Institution
			delivered := false
			for _, id := range e.institutions[p.Institution].Members {
				if id == p.ID || !e.player(id).Alive {
					continue
				}
				if m.To == "institution" || m.To == id {
					if len(next[id]) < 64 {
						next[id] = append(next[id], m)
						delivered = true
					}
				}
			}
			if delivered {
				if m.To == "institution" {
					e.publicMessages = append(e.publicMessages, m)
					if len(e.publicMessages) > 50 {
						e.publicMessages = e.publicMessages[len(e.publicMessages)-50:]
					}
				}
				p.Brain -= e.config.MessageCost
			}
		}
	}
	e.inbox = next
}
