// Package game defines the public agent contract and authoritative simulation.
package game

import (
	"encoding/json"
	"errors"
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
	"math"
	"math/rand"
)

type Config struct {
	Seed                int64      `json:"seed"`
	Agents              int        `json:"agents"`
	Institutions        int        `json:"institutions"`
	Rounds              int        `json:"rounds"`
	Cycles              int        `json:"cycles"`
	Turns               int        `json:"turns"` // 0 means one harvest per living member per round
	ReformEvery         int        `json:"reform_every"`
	PoliticalPasses     int        `json:"political_passes"`
	InitialPatch        float64    `json:"initial_patch"`
	Capacity            float64    `json:"capacity"`
	InitialBrain        float64    `json:"initial_brain"`
	MaxBrain            float64    `json:"max_brain"`
	InitialBody         float64    `json:"initial_body"`
	MaxBody             float64    `json:"max_body"`
	Sun                 float64    `json:"sun"`
	Cloud               float64    `json:"cloud"`
	Metabolism          float64    `json:"metabolism"`
	MaxEat              float64    `json:"max_eat"`
	FoodToBody          float64    `json:"food_to_body"`
	Rot                 float64    `json:"rot"`
	HarvestCost         float64    `json:"harvest_cost"`
	ProposalCost        float64    `json:"proposal_cost"`
	VoteCost            float64    `json:"vote_cost"`
	MessageCost         float64    `json:"message_cost"`
	MinContribution     float64    `json:"min_contribution"`
	Thresholds          [4]float64 `json:"thresholds"`
	YieldBase           float64    `json:"yield_base"`
	YieldSlope          float64    `json:"yield_slope"`
	YieldCap            int        `json:"yield_cap"`
	YieldExtra          float64    `json:"yield_extra"`
	BrainUtility        float64    `json:"brain_utility"`
	MinimumResidual     float64    `json:"minimum_residual"`
	Revive              bool       `json:"revive"`
	CarryMemory         bool       `json:"carry_memory"`
	PublicContributions bool       `json:"public_contributions"`
	Strategies          []string   `json:"strategies"`
}

func DefaultConfig() Config {
	return Config{Seed: 42, Agents: 10, Institutions: 2, Rounds: 30, Cycles: 1, ReformEvery: 10, PoliticalPasses: 3, InitialPatch: 30, Capacity: 60, InitialBrain: 3, MaxBrain: 10, InitialBody: 6, MaxBody: 12, Sun: 1.5, Cloud: .5, Metabolism: .7, MaxEat: 2, FoodToBody: 1, Rot: .02, HarvestCost: .1, ProposalCost: .05, VoteCost: .02, MessageCost: .01, MinContribution: .5, Thresholds: [4]float64{1, 2.5, 4, 5}, YieldBase: .5, YieldSlope: .2, YieldCap: 5, YieldExtra: 1, BrainUtility: .05, Revive: true, Strategies: []string{"random"}}
}
func (c Config) Validate() error {
	b, _ := json.Marshal(c)
	if b == nil {
		return errors.New("non-finite configuration")
	}
	if c.Agents < 1 || c.Agents > 200 || c.Institutions < 1 || c.Institutions > c.Agents || c.Rounds < 1 || c.Rounds > 2000 || c.Cycles < 1 || c.Cycles > 100 || c.Rounds*c.Cycles > 10000 || c.Turns < 0 || c.Turns > 1000 || c.ReformEvery < 1 || c.PoliticalPasses < 1 || c.PoliticalPasses > 10 {
		return errors.New("invalid population or timing limits")
	}
	nums := []float64{c.InitialPatch, c.InitialBrain, c.InitialBody, c.Sun, c.Cloud, c.Metabolism, c.MaxEat, c.Rot, c.HarvestCost, c.ProposalCost, c.VoteCost, c.MessageCost, c.MinContribution, c.YieldBase, c.YieldSlope, c.YieldExtra, c.BrainUtility, c.MinimumResidual}
	for _, n := range nums {
		if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > 1e6 {
			return errors.New("resource parameters must be finite and nonnegative (<= 1e6)")
		}
	}
	if c.Capacity <= 0 || c.Capacity > 1e6 || c.MaxBrain <= 0 || c.MaxBrain > 1e6 || c.MaxBody <= 0 || c.MaxBody > 1e6 || c.FoodToBody <= 0 || c.FoodToBody > 100 || c.InitialPatch > c.Capacity || c.InitialBrain > c.MaxBrain || c.InitialBody > c.MaxBody || c.Cloud > 1 || c.Rot > 1 || c.YieldCap < 1 || c.YieldCap > 200 || c.MinContribution <= 0 || c.MinimumResidual > c.Capacity {
		return errors.New("invalid resource capacity or fraction")
	}
	for i, t := range c.Thresholds {
		if math.IsNaN(t) || math.IsInf(t, 0) || t < 0 || t > 1e6 || (i > 0 && t < c.Thresholds[i-1]) {
			return errors.New("thresholds must be finite, nonnegative and ordered")
		}
	}
	if len(c.Strategies) == 0 {
		return errors.New("at least one strategy required")
	}
	return nil
}

type Player struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Strategy     string  `json:"strategy"`
	Institution  int     `json:"institution"`
	Brain        float64 `json:"brain"`
	Body         float64 `json:"body"`
	Stock        float64 `json:"stock"`
	Alive        bool    `json:"alive"`
	Utility      float64 `json:"utility"`
	Violations   int     `json:"violations"`
	LastAction   string  `json:"last_action"`
	Contribution float64 `json:"contribution"`
}
type Rules struct {
	Quota         float64 `json:"quota"`
	Order         string  `json:"order"`
	VoteThreshold float64 `json:"vote_threshold"`
}
type Institution struct {
	ID           int               `json:"id"`
	Members      []string          `json:"members"`
	CPR          float64           `json:"cpr"`
	CTK          float64           `json:"ctk"`
	LastCTK      float64           `json:"last_ctk"`
	Wasted       float64           `json:"wasted"`
	Contributors int               `json:"contributors"`
	Regrowth     float64           `json:"regrowth"`
	Rules        Rules             `json:"rules"`
	Order        []string          `json:"order"`
	Shared       map[string]string `json:"shared"`
}
type Message struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Round       int    `json:"round"`
	Institution int    `json:"institution"`
}
type Proposal struct {
	Key   string  `json:"key"`
	Value float64 `json:"value"`
	Order string  `json:"order"`
}
type Decision struct {
	Harvest    float64
	Eat        float64
	Contribute float64
}
type VoteRecord struct {
	Round       int      `json:"round"`
	Institution int      `json:"institution"`
	Proposer    string   `json:"proposer"`
	Proposal    Proposal `json:"proposal"`
	Yes         int      `json:"yes"`
	Cast        int      `json:"cast"`
	Accepted    bool     `json:"accepted"`
}
type Metrics struct {
	Utility           float64 `json:"utility"`
	AverageUtility    float64 `json:"average_utility"`
	CPR               float64 `json:"cpr"`
	Alive             int     `json:"alive"`
	CTK               float64 `json:"ctk"`
	Contributors      int     `json:"contributors"`
	FreeRiders        int     `json:"free_riders"`
	Cooperation       float64 `json:"cooperation"`
	Sustainability    float64 `json:"sustainability"`
	Accepted          int     `json:"accepted"`
	Violations        int     `json:"violations"`
	MembershipChanges int     `json:"membership_changes"`
}
type Snapshot struct {
	PublicMessages []Message     `json:"public_messages"`
	Round          int           `json:"round"`
	Cycle          int           `json:"cycle"`
	Phase          string        `json:"phase"`
	Players        []Player      `json:"players"`
	Institutions   []Institution `json:"institutions"`
	Votes          []VoteRecord  `json:"votes"`
	Events         []string      `json:"events"`
	Metrics        Metrics       `json:"metrics"`
}

// Memory capabilities are bound to one agent by the engine; no owner argument is accepted.
type Memory interface {
	Get(string) string
	Set(string, string)
}
type Observation struct {
	Self         Player
	Peers        []PublicAgent
	Institutions []Institution
	Institution  Institution
	Inbox        []Message
	Round        int
	Config       Config
}
type PublicAgent struct {
	ID           string
	Institution  int
	Alive        bool
	Contribution *float64
}

// Agent embeds the actual university v2 interface. Use Base to supply its plumbing.
type Agent interface {
	agent.IAgent[Agent]
	Strategy() string
	ChooseInstitution(Observation, Memory, *rand.Rand) int // -1 accepts seeded assignment
	Propose(Observation, Memory, *rand.Rand) *Proposal
	Vote(Observation, Proposal, Memory, *rand.Rand) bool
	Act(Observation, Memory, *rand.Rand) Decision
	Communicate(Observation, Memory, *rand.Rand) []Message
}
type Base struct {
	*agent.BaseAgent[Agent]
	id uuid.UUID
}

func NewBase(s agent.IExposedServerFunctions[Agent]) *Base {
	b := &Base{BaseAgent: agent.CreateBaseAgent[Agent](s)}
	if own, ok := s.(interface{ Identity() uuid.UUID }); ok {
		b.id = own.Identity()
	} else {
		b.id = b.BaseAgent.GetID()
	}
	return b
}
func (*Base) ChooseInstitution(Observation, Memory, *rand.Rand) int { return -1 }
func (*Base) Propose(Observation, Memory, *rand.Rand) *Proposal     { return nil }
func (*Base) Vote(Observation, Proposal, Memory, *rand.Rand) bool   { return false }
func (*Base) Act(Observation, Memory, *rand.Rand) Decision          { return Decision{} }
func (*Base) Communicate(Observation, Memory, *rand.Rand) []Message { return nil }

type Factory func(agent.IExposedServerFunctions[Agent]) Agent
type Registry map[string]Factory

func (r Registry) Register(name string, f Factory) {
	if name == "" || f == nil || r[name] != nil {
		panic("invalid or duplicate strategy")
	}
	r[name] = f
}

func (b *Base) GetID() uuid.UUID { return b.id }
