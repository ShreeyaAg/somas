package api

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"somas/internal/store"
	"somas/pkg/game"
	"sort"
	"strings"
	"sync"
	"time"
)

type liveRun struct {
	record   store.Run
	engine   *game.Engine
	scope    string
	seq      int
	interval time.Duration
	next     time.Time
	initial  map[string]map[string]string
}
type Server struct {
	mu       sync.Mutex
	db       *store.Store
	registry game.Registry
	tokens   map[string]string
	live     map[string]*liveRun
	handler  http.Handler
	stop     chan struct{}
	done     chan struct{}
}
type NewRequest struct {
	Config          game.Config `json:"config"`
	MemoryNamespace string      `json:"memory_namespace"`
}

func New(db *store.Store, reg game.Registry, tokens map[string]string, assets http.Handler) *Server {
	s := &Server{db: db, registry: reg, tokens: tokens, live: map[string]*liveRun{}, stop: make(chan struct{}), done: make(chan struct{})}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", s.api)
	mux.Handle("/", assets)
	s.handler = mux
	go s.loop()
	return s
}
func (s *Server) Close() { close(s.stop); <-s.done }
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; object-src 'none'; frame-ancestors 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	s.handler.ServeHTTP(w, r)
}
func (s *Server) owner(r *http.Request) string {
	value := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	given := sha256.Sum256([]byte(value))
	for group, key := range s.tokens {
		expected := sha256.Sum256([]byte(key))
		if subtle.ConstantTimeCompare(given[:], expected[:]) == 1 {
			return group
		}
	}
	return ""
}
func write(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 65536)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return errors.New("one JSON object required")
	}
	return nil
}
func (s *Server) loop() {
	defer close(s.done)
	t := time.NewTicker(50 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case now := <-t.C:
			s.mu.Lock()
			for _, run := range s.live {
				if run.record.Playing && !now.Before(run.next) {
					s.advance(run)
					run.next = now.Add(run.interval)
				}
			}
			s.mu.Unlock()
		}
	}
}
func (s *Server) advance(run *liveRun) {
	defer func() {
		if x := recover(); x != nil {
			run.record.Playing = false
			run.record.Error = fmt.Sprintf("strategy panic: %v", x)
			run.seq++
			_ = s.db.Save(run.record, run.scope, run.engine.Memories(), run.seq)
		}
	}()
	if !run.engine.Step() {
		run.record.Playing = false
		return
	}
	run.record.State = run.engine.Snapshot()
	if run.record.State.Phase == "complete" {
		run.record.Playing = false
	}
	run.seq++
	if err := s.db.Save(run.record, run.scope, run.engine.Memories(), run.seq); err != nil {
		run.record.Playing = false
		run.record.Error = "persistence failed: " + err.Error()
	}
}
func randomID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func (s *Server) create(owner string, q NewRequest) (store.Run, error) {
	if err := q.Config.Validate(); err != nil {
		return store.Run{}, err
	}
	count := 0
	for _, r := range s.live {
		if r.record.Group == owner {
			count++
		}
	}
	if count >= 30 {
		return store.Run{}, errors.New("active run limit reached; restart server to archive completed runs")
	}
	id := randomID()
	scope := "run:" + id
	if q.Config.CarryMemory {
		if len(q.MemoryNamespace) < 1 || len(q.MemoryNamespace) > 64 {
			return store.Run{}, errors.New("carry_memory requires a memory_namespace (1–64 characters)")
		}
		scope = "carry:" + q.MemoryNamespace
		for _, r := range s.live {
			if r.record.Group == owner && r.scope == scope && r.record.State.Phase != "complete" {
				return store.Run{}, errors.New("memory namespace is already in use")
			}
		}
	}
	mem, err := s.db.Memories(owner, scope)
	if err != nil {
		return store.Run{}, err
	}
	e, err := game.New(q.Config, s.registry, mem)
	if err != nil {
		return store.Run{}, err
	}
	r := store.Run{ID: id, Group: owner, Config: q.Config, State: e.Snapshot()}
	if err = s.db.Save(r, scope, e.Memories(), 0); err != nil {
		return store.Run{}, err
	}
	s.live[id] = &liveRun{record: r, engine: e, scope: scope, interval: 500 * time.Millisecond, initial: mem}
	return r, nil
}
func (s *Server) api(w http.ResponseWriter, r *http.Request) {
	owner := s.owner(r)
	if owner == "" {
		fail(w, 401, "valid group bearer token required")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	// All state changes are serialized with persistence; browser polling is race-free.
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() {
		if x := recover(); x != nil {
			fail(w, 500, fmt.Sprintf("strategy or server panic: %v", x))
		}
	}()
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/"), "/")
	parts := strings.Split(path, "/")
	if path == "defaults" && r.Method == "GET" {
		write(w, game.DefaultConfig())
		return
	}
	if path == "strategies" && r.Method == "GET" {
		names := []string{}
		for k := range s.registry {
			names = append(names, k)
		}
		sort.Strings(names)
		write(w, names)
		return
	}
	if path == "runs" {
		switch r.Method {
		case "GET":
			rs, err := s.db.Runs(owner)
			if err != nil {
				fail(w, 500, err.Error())
				return
			}
			for i := range rs {
				if live := s.live[rs[i].ID]; live != nil {
					rs[i] = live.record
				}
			}
			write(w, rs)
			return
		case "POST":
			var q NewRequest
			if err := decode(w, r, &q); err != nil {
				fail(w, 400, err.Error())
				return
			}
			run, err := s.create(owner, q)
			if err != nil {
				fail(w, 400, err.Error())
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(201)
			write(w, run)
			return
		}
	}
	if path == "memory/reset" && r.Method == "POST" {
		var q struct {
			Namespace string `json:"namespace"`
		}
		if err := decode(w, r, &q); err != nil || q.Namespace == "" {
			fail(w, 400, "namespace required")
			return
		}
		scope := "carry:" + q.Namespace
		for _, run := range s.live {
			if run.record.Group == owner && run.scope == scope && run.record.State.Phase != "complete" {
				fail(w, 409, "finish the run before resetting its memory")
				return
			}
		}
		if err := s.db.ResetMemory(owner, scope); err != nil {
			fail(w, 500, err.Error())
			return
		}
		write(w, map[string]bool{"reset": true})
		return
	}
	if len(parts) >= 2 && parts[0] == "runs" {
		id := parts[1]
		rs, err := s.db.Runs(owner)
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		var saved *store.Run
		for i := range rs {
			if rs[i].ID == id {
				saved = &rs[i]
				break
			}
		}
		if saved == nil {
			fail(w, 404, "run not found")
			return
		}
		live := s.live[id]
		if live != nil {
			saved = &live.record
		}
		if len(parts) == 2 && r.Method == "GET" {
			write(w, saved)
			return
		}
		if len(parts) == 3 && r.Method == "GET" && (parts[2] == "history" || parts[2] == "export.json" || parts[2] == "export.csv") {
			h, err := s.db.History(owner, id)
			if err != nil {
				fail(w, 500, err.Error())
				return
			}
			if parts[2] == "export.csv" {
				w.Header().Set("Content-Type", "text/csv")
				w.Header().Set("Content-Disposition", "attachment; filename=experiment.csv")
				cw := csv.NewWriter(w)
				_ = cw.Write([]string{"round", "cycle", "phase", "utility", "average_utility", "cpr", "alive", "ctk", "contributors", "free_riders", "cooperation", "sustainability", "accepted", "violations", "membership_changes"})
				for _, st := range h {
					m := st.Metrics
					_ = cw.Write([]string{fmt.Sprint(st.Round), fmt.Sprint(st.Cycle), st.Phase, fmt.Sprint(m.Utility), fmt.Sprint(m.AverageUtility), fmt.Sprint(m.CPR), fmt.Sprint(m.Alive), fmt.Sprint(m.CTK), fmt.Sprint(m.Contributors), fmt.Sprint(m.FreeRiders), fmt.Sprint(m.Cooperation), fmt.Sprint(m.Sustainability), fmt.Sprint(m.Accepted), fmt.Sprint(m.Violations), fmt.Sprint(m.MembershipChanges)})
				}
				cw.Flush()
				return
			}
			if parts[2] == "export.json" {
				write(w, map[string]any{"config": saved.Config, "history": h})
				return
			}
			write(w, h)
			return
		}
		if len(parts) == 3 && parts[2] == "control" && r.Method == "POST" {
			if live == nil {
				fail(w, 409, "archived run: replay history or create a new run")
				return
			}
			var q struct {
				Action string `json:"action"`
				Speed  int    `json:"speed_ms"`
			}
			if err := decode(w, r, &q); err != nil {
				fail(w, 400, err.Error())
				return
			}
			switch q.Action {
			case "play":
				if live.record.Error != "" {
					fail(w, 409, "run has an error; reset first")
					return
				}
				live.record.Playing = true
				live.next = time.Now()
			case "archive":
				if live.record.State.Phase != "complete" {
					fail(w, 409, "only completed runs can be archived")
					return
				}
				live.record.Playing = false
				delete(s.live, id)
			case "pause":
				live.record.Playing = false
			case "step":
				live.record.Playing = false
				s.advance(live)
			case "reset":
				for _, other := range s.live {
					if other != live && other.record.Group == owner && other.scope == live.scope && other.record.State.Phase != "complete" {
						fail(w, 409, "memory namespace is in use")
						return
					}
				}
				e, err := game.New(live.record.Config, s.registry, live.initial)
				if err != nil {
					fail(w, 400, err.Error())
					return
				}
				live.engine = e
				live.record.Playing = false
				live.record.Error = ""
				live.record.State = e.Snapshot()
				live.seq++
				if err = s.db.Save(live.record, live.scope, e.Memories(), live.seq); err != nil {
					fail(w, 500, err.Error())
					return
				}
			case "speed":
				if q.Speed < 50 || q.Speed > 10000 {
					fail(w, 400, "speed_ms must be 50–10000")
					return
				}
				live.interval = time.Duration(q.Speed) * time.Millisecond
			default:
				fail(w, 400, "unknown action")
				return
			}
			write(w, live.record)
			return
		}
	}
	fail(w, 404, "endpoint not found")
}
