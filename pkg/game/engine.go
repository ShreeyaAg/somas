package game

import (
	"encoding/json"
	"fmt"
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/message"
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/server"
	"github.com/google/uuid"
	"math/rand"
	"time"
)

type Engine struct {
	config         Config
	rng            *rand.Rand
	round          int
	phase          string
	players        []Player
	institutions   []Institution
	agents         map[string]Agent
	memories       map[string]map[string]string
	inbox          map[string][]Message
	events         []string
	publicMessages []Message
	votes          []VoteRecord
	changes        int
	platform       *server.BaseServer[Agent]
}
type localMemory struct{ data map[string]string }

func (m localMemory) Get(k string) string { return m.data[k] }
func (m localMemory) Set(k, v string) {
	if len(k) > 0 && len(k) <= 128 && len(v) <= 8192 && (len(m.data) < 256 || m.data[k] != "") {
		m.data[k] = v
	}
}
func clone[T any](v T) T { b, _ := json.Marshal(v); var out T; _ = json.Unmarshal(b, &out); return out }
func New(c Config, registry Registry, memory map[string]map[string]string) (*Engine, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	for _, s := range c.Strategies {
		if registry[s] == nil {
			return nil, fmt.Errorf("unknown strategy %q", s)
		}
	}
	e := &Engine{config: clone(c), rng: rand.New(rand.NewSource(c.Seed)), phase: "formation", agents: map[string]Agent{}, memories: clone(memory), inbox: map[string][]Message{}}
	if e.memories == nil {
		e.memories = map[string]map[string]string{}
	}
	e.platform = server.CreateBaseServer[Agent](c.Rounds*c.Cycles, 1, time.Millisecond, 16)
	for i := 0; i < c.Agents; i++ {
		id := fmt.Sprintf("agent-%03d", i+1)
		name := c.Strategies[i%len(c.Strategies)]
		stable := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%d/%s", c.Seed, id)))
		a := registry[name](&restrictedPlatform{IExposedServerFunctions: e.platform, id: stable})
		e.platform.AddAgent(a)
		e.agents[id] = a
		e.players = append(e.players, Player{ID: id, Name: id, Strategy: name, Institution: -1, Brain: c.InitialBrain, Body: c.InitialBody, Alive: c.InitialBody > 0})
		if e.memories[id] == nil {
			e.memories[id] = map[string]string{}
		}
	}
	e.platform.SetGameRunner(e)
	e.form()
	return e, nil
}
func (e *Engine) Config() Config                         { return clone(e.config) }
func (e *Engine) Memories() map[string]map[string]string { return clone(e.memories) }
func (e *Engine) Memory(id string) Memory                { return localMemory{e.memories[id]} }
func (e *Engine) note(s string) {
	e.events = append(e.events, fmt.Sprintf("R%d %s", e.round, s))
	if len(e.events) > 150 {
		e.events = e.events[len(e.events)-150:]
	}
}
func (e *Engine) player(id string) *Player {
	for i := range e.players {
		if e.players[i].ID == id {
			return &e.players[i]
		}
	}
	return nil
}
func (e *Engine) observe(id string) Observation {
	p := e.player(id)
	v := Observation{Self: *p, Round: e.round, Config: clone(e.config), Inbox: clone(e.inbox[id]), Institutions: clone(e.institutions)}
	if p.Institution >= 0 {
		v.Institution = clone(e.institutions[p.Institution])
	}
	for _, other := range e.players {
		pub := PublicAgent{ID: other.ID, Institution: other.Institution, Alive: other.Alive}
		if e.config.PublicContributions {
			x := other.Contribution
			pub.Contribution = &x
		}
		v.Peers = append(v.Peers, pub)
	}
	return v
}
func (e *Engine) form() {
	e.institutions = nil
	for i := 0; i < e.config.Institutions; i++ {
		e.institutions = append(e.institutions, Institution{ID: i, CPR: e.config.InitialPatch, Rules: Rules{Quota: 2, Order: "random", VoteThreshold: .5}, Shared: map[string]string{}})
	}
	perm := e.rng.Perm(len(e.players))
	for n, i := range perm {
		p := &e.players[i]
		if !p.Alive && e.config.Revive {
			p.Alive = true
			p.Body = e.config.InitialBody
			p.Brain = e.config.InitialBrain
			p.Stock = 0
		}
		if !p.Alive {
			continue
		}
		g := n % len(e.institutions)
		choice := e.agents[p.ID].ChooseInstitution(e.observe(p.ID), e.Memory(p.ID), e.rng)
		if choice >= 0 && choice < len(e.institutions) {
			g = choice
		}
		if p.Institution != g {
			e.changes++
		}
		p.Institution = g
		e.institutions[g].Members = append(e.institutions[g].Members, p.ID)
		e.Memory(p.ID).Set("last_institution", fmt.Sprint(g))
	}
	e.note("institutions formed; patches and AVAs reset")
}

// Step advances one observable phase. Only the engine owns mutable authoritative state.
func (e *Engine) Step() bool {
	switch e.phase {
	case "complete":
		return false
	case "formation", "round_end":
		if e.round >= e.config.Rounds*e.config.Cycles {
			e.phase = "complete"
			return true
		}
		if e.round > 0 && (e.round%e.config.ReformEvery == 0 || e.round%e.config.Rounds == 0) {
			e.form()
		}
		e.round++
		for i := range e.players {
			p := &e.players[i]
			p.Contribution = 0
			if p.Alive {
				p.Brain = min(e.config.MaxBrain, p.Brain+e.config.Sun*(1-e.rng.Float64()*e.config.Cloud))
			}
		}
		e.phase = "political"
		e.politics()
	case "political":
		e.communicate()
		e.phase = "operational"
	case "operational":
		e.operate()
		e.phase = "round_end"
	}
	return true
}
func (e *Engine) Snapshot() Snapshot {
	s := Snapshot{PublicMessages: clone(e.publicMessages), Round: e.round, Cycle: 1 + max(0, e.round-1)/e.config.Rounds, Phase: e.phase, Players: clone(e.players), Institutions: clone(e.institutions), Votes: clone(e.votes), Events: clone(e.events)}
	for _, p := range e.players {
		s.Metrics.Utility += p.Utility
		s.Metrics.Violations += p.Violations
		if p.Alive {
			s.Metrics.Alive++
		}
	}
	for _, g := range e.institutions {
		s.Metrics.CPR += g.CPR
		s.Metrics.CTK += g.LastCTK
		s.Metrics.Contributors += g.Contributors
	}
	for _, v := range e.votes {
		if v.Accepted {
			s.Metrics.Accepted++
		}
	}
	participants := 0
	for _, g := range e.institutions {
		participants += len(g.Order)
	}
	s.Metrics.FreeRiders = max(0, participants-s.Metrics.Contributors)
	if participants > 0 {
		s.Metrics.Cooperation = float64(s.Metrics.Contributors) / float64(participants)
	}
	s.Metrics.AverageUtility = s.Metrics.Utility / float64(len(e.players))
	s.Metrics.Sustainability = s.Metrics.CPR / (e.config.Capacity * float64(len(e.institutions)))
	s.Metrics.MembershipChanges = e.changes
	return s
}
func (e *Engine) Run() {
	for e.Step() {
	}
}

// University lifecycle adapter: callbacks and dashboard Step share the same engine.
func (e *Engine) RunStartOfIteration(int) {
	if e.phase == "formation" || e.phase == "round_end" {
		e.Step()
	}
}
func (e *Engine) RunTurn(int, int) {
	if e.phase == "political" {
		e.Step()
	}
	if e.phase == "operational" {
		e.Step()
	}
}
func (e *Engine) RunEndOfIteration(int) {}

var _ server.GameRunner = (*Engine)(nil)

// Restrict the framework facade: no direct references to peer implementations and
// no bypass of the engine's scoped, deterministic typed-message protocol.
type restrictedPlatform struct {
	agent.IExposedServerFunctions[Agent]
	id uuid.UUID
}

func (s *restrictedPlatform) Identity() uuid.UUID             { return s.id }
func (s *restrictedPlatform) AccessAgentByID(uuid.UUID) Agent { return nil }
func (s *restrictedPlatform) ViewAgentIdSet() map[uuid.UUID]struct{} {
	return map[uuid.UUID]struct{}{s.id: {}}
}
func (s *restrictedPlatform) DeliverMessage(message.IMessage[Agent], uuid.UUID) {}
func (s *restrictedPlatform) AgentStoppedTalking(uuid.UUID)                     {}

// RunWithPlatform drives the same engine through the actual v2 Start lifecycle.
// Start only once on a new Engine. Typed messaging remains engine-managed.
func (e *Engine) RunWithPlatform() {
	e.platform.Start()
	if e.phase == "round_end" {
		e.Step()
	}
}
