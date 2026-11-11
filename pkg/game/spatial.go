package game

import (
	"fmt"
	"math"
	"math/rand"
)

// NormalizeConfig derives a deterministic world from population. Grid dimensions
// are minimum sizes: growth never removes or overlaps a patch.
func NormalizeConfig(c Config) (Config, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	if c.AgentsPerPatch == 0 {
		c.AgentsPerPatch = 5
	}
	if c.AutoPatches {
		c.Institutions = (c.Agents + c.AgentsPerPatch - 1) / c.AgentsPerPatch
	}
	if c.GridWidth == 0 {
		c.GridWidth = 12
	}
	if c.GridHeight == 0 {
		c.GridHeight = 9
	}
	c.GridWidth = max(6, c.GridWidth)
	c.GridHeight = max(6, c.GridHeight)
	for (c.GridWidth/3)*(c.GridHeight/3) < c.Institutions {
		if c.GridWidth <= c.GridHeight && c.GridWidth < 64 {
			c.GridWidth = min(64, c.GridWidth+3)
		} else if c.GridHeight < 64 {
			c.GridHeight = min(64, c.GridHeight+3)
		} else if c.GridWidth < 64 {
			c.GridWidth = min(64, c.GridWidth+3)
		} else {
			return c, fmt.Errorf("population exceeds world capacity")
		}
	}
	if c.HarvestAccess == "" {
		c.HarvestAccess = "members"
	}
	return c, c.Validate()
}
func (e *Engine) layoutWorld() {
	e.world = World{Width: e.config.GridWidth, Height: e.config.GridHeight}
	columns := min(e.world.Width/3, max(int(math.Ceil(math.Sqrt(float64(len(e.institutions))))), (len(e.institutions)+e.world.Height/3-1)/(e.world.Height/3)))
	rows := (len(e.institutions) + columns - 1) / columns
	offsetX := (e.world.Width - columns*3) / 2
	offsetY := (e.world.Height - rows*3) / 2
	for _, g := range e.institutions {
		e.world.Patches = append(e.world.Patches, Patch{ID: g.ID, Institution: g.ID, Origin: Position{X: offsetX + 1 + 3*(g.ID%columns), Y: offsetY + 1 + 3*(g.ID/columns)}, Width: 2, Height: 2})
	}
	// Formation/reformation places members on their home patch; movement thereafter
	// changes only their location, never membership, CPR ownership, or CTK destination.
	for i := range e.players {
		p := &e.players[i]
		if p.Institution < 0 {
			continue
		}
		patch := e.world.Patches[p.Institution]
		slot := 0
		for _, id := range e.institutions[p.Institution].Members {
			if id == p.ID {
				break
			}
			slot++
		}
		p.Position = Position{X: patch.Origin.X + slot%2, Y: patch.Origin.Y + (slot/2)%2}
	}
	if e.config.Spatial {
		e.seedClouds()
		e.paintClouds()
	}
}
// cloudBlob is one drifting cloud: a soft disc of radius R and peak density D.
type cloudBlob struct{ X, Y, R, D float64 }

func (e *Engine) newCloud(initial bool) cloudBlob {
	w, h := float64(e.world.Width), float64(e.world.Height)
	c := cloudBlob{R: 1.5 + 2*e.rng.Float64(), Y: e.rng.Float64() * h, D: .5 + .5*e.rng.Float64()}
	if initial {
		c.X = e.rng.Float64() * w
	} else {
		c.X = -c.R
	}
	return c
}

// seedClouds creates the cloud population once per experiment (it survives re-formation).
func (e *Engine) seedClouds() {
	if len(e.clouds) > 0 || e.world.Width == 0 {
		return
	}
	for i := 0; i < max(2, e.world.Width*e.world.Height/30); i++ {
		e.clouds = append(e.clouds, e.newCloud(true))
	}
}

// driftClouds moves every cloud (wind blows east with some wobble) and repaints the field.
func (e *Engine) driftClouds() {
	e.seedClouds()
	w, h := float64(e.world.Width), float64(e.world.Height)
	for i := range e.clouds {
		c := &e.clouds[i]
		c.X += 1 + e.rng.Float64() - .5
		c.Y += e.rng.Float64() - .5
		c.R = Clamp(c.R+(e.rng.Float64()-.5)*.6, 1.5, 3.5)
		c.D = Clamp(c.D+(e.rng.Float64()-.5)*.3, .3, 1)
		if c.X-c.R > w || c.Y < -c.R || c.Y > h+c.R {
			*c = e.newCloud(false)
		}
	}
	e.paintClouds()
}

// paintClouds rasterises the blobs into World.Clouds without consuming randomness.
func (e *Engine) paintClouds() {
	w, h := e.world.Width, e.world.Height
	e.world.Clouds = make([]float64, w*h)
	if e.config.Cloud <= 0 {
		return
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sum := 0.0
			for _, c := range e.clouds {
				dx, dy := float64(x)+.5-c.X, float64(y)+.5-c.Y
				if d := math.Sqrt(dx*dx + dy*dy); d < c.R {
					sum += c.D * (1 - d/c.R)
				}
			}
			e.world.Clouds[y*w+x] = math.Round(Clamp(sum, 0, 1)*e.config.Cloud*1000) / 1000
		}
	}
}

// cloudFor returns the cloud cover that blocks this player's sunlight this round.
func (e *Engine) cloudFor(p Player) float64 {
	if e.config.Spatial {
		w := e.world.Width
		if i := p.Position.Y*w + p.Position.X; p.Position.X >= 0 && p.Position.X < w && i >= 0 && i < len(e.world.Clouds) {
			return e.world.Clouds[i]
		}
		return 0
	}
	return e.rng.Float64() * e.config.Cloud
}
func (p Patch) Contains(pos Position) bool {
	return pos.X >= p.Origin.X && pos.X < p.Origin.X+p.Width && pos.Y >= p.Origin.Y && pos.Y < p.Origin.Y+p.Height
}
func (e *Engine) patchAt(pos Position) *Patch {
	for i := range e.world.Patches {
		if e.world.Patches[i].Contains(pos) {
			return &e.world.Patches[i]
		}
	}
	return nil
}
func distance(a, b Position) int {
	dx := a.X - b.X
	if dx < 0 {
		dx = -dx
	}
	dy := a.Y - b.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}
func (e *Engine) moveAgents() {
	if !e.config.Spatial {
		return
	}
	for i := range e.players {
		p := &e.players[i]
		if !p.Alive {
			continue
		}
		p.LastAction = "stay"
		if p.Activity != nil {
			p.Activity.MoveStatus = "stayed"
		}
		mover, ok := e.agents[p.ID].(Mover)
		if !ok {
			continue
		}
		to := mover.Move(e.observe(p.ID), e.Memory(p.ID), e.rng)
		// Check bounds before subtracting coordinates so adversarial int extremes
		// cannot overflow the Manhattan-distance validation.
		if to.X < 0 || to.Y < 0 || to.X >= e.world.Width || to.Y >= e.world.Height {
			if p.Activity != nil {
				p.Activity.MoveStatus = "outside world"
			}
			e.note(p.ID + " move rejected: outside world")
			continue
		}
		if to == p.Position {
			continue
		}
		if distance(p.Position, to) != 1 {
			if p.Activity != nil {
				p.Activity.MoveStatus = "one cardinal step required"
			}
			e.note(p.ID + " move rejected: one cardinal step required")
			continue
		}
		if p.Brain < e.config.MoveBrainCost || p.Body < e.config.MoveBodyCost {
			if p.Activity != nil {
				p.Activity.MoveStatus = "insufficient resources"
			}
			e.note(p.ID + " move rejected: insufficient resources")
			continue
		}
		from := p.Position
		p.Position = to
		p.Moves++
		p.Brain = max(0, p.Brain-e.config.MoveBrainCost)
		p.Body = max(0, p.Body-e.config.MoveBodyCost)
		if p.Activity != nil {
			p.Activity.MoveStatus = "moved"
			p.Activity.Moves++
			p.Activity.MoveBrainCost += e.config.MoveBrainCost
			p.Activity.MoveBodyCost += e.config.MoveBodyCost
		}
		p.LastAction = fmt.Sprintf("move (%d,%d) → (%d,%d)", from.X, from.Y, to.X, to.Y)
		if p.Body == 0 {
			p.Alive = false
			e.note(p.ID + " died moving")
		}
	}
}

// StepToward returns a single cardinal step toward a target, or stays at target.
// Policies may use this helper; the server still validates every requested move.
func StepToward(from, to Position) Position {
	if from.X < to.X {
		from.X++
	} else if from.X > to.X {
		from.X--
	} else if from.Y < to.Y {
		from.Y++
	} else if from.Y > to.Y {
		from.Y--
	}
	return from
}
func Wander(o Observation, r *rand.Rand) Position {
	p := o.Self.Position
	switch r.Intn(5) {
	case 0:
		p.X++
	case 1:
		p.X--
	case 2:
		p.Y++
	case 3:
		p.Y--
	}
	if p.X < 0 || p.Y < 0 || p.X >= o.World.Width || p.Y >= o.World.Height {
		return o.Self.Position
	}
	return p
}
