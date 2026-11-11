package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"somas/internal/store"
	"somas/pkg/agents"
	"somas/pkg/game"
	"somas/web"
	"strings"
	"testing"
)

func TestAPIIsolationControlsAndExport(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db, agents.Registry(), map[string]string{"a": "a-token", "b": "b-token"}, web.Handler())
	defer s.Close()
	call := func(method, path, token string, body any) *httptest.ResponseRecorder {
		t.Helper()
		var b bytes.Buffer
		if body != nil {
			if err := json.NewEncoder(&b).Encode(body); err != nil {
				t.Fatal(err)
			}
		}
		r := httptest.NewRequest(method, path, &b)
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	if w := call("GET", "/api/runs", "", nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	c := game.DefaultConfig()
	c.Rounds = 2
	c.Strategies = []string{"cooperative"}
	w := call("POST", "/api/runs", "a-token", NewRequest{Config: c})
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var run store.Run
	if err = json.Unmarshal(w.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"", "/history", "/export.csv", "/export.json"} {
		if w = call("GET", "/api/runs/"+run.ID+endpoint, "b-token", nil); w.Code != 404 {
			t.Fatal("cross group leak", endpoint, w.Code)
		}
	}
	for i := 0; i < 7; i++ {
		w = call("POST", "/api/runs/"+run.ID+"/control", "a-token", map[string]string{"action": "step"})
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
	}
	w = call("GET", "/api/runs/"+run.ID+"/export.json", "a-token", nil)
	if strings.Contains(w.Body.String(), "decisions") {
		t.Fatal("private memory or inbox leaked")
	}
	if !strings.Contains(w.Body.String(), "complete") {
		t.Fatal("run incomplete")
	}
	w = call("GET", "/api/runs/"+run.ID+"/export.csv", "a-token", nil)
	if !strings.Contains(w.Body.String(), "cooperation") {
		t.Fatal("export missing")
	}
	if w = call("POST", "/api/runs/"+run.ID+"/control", "a-token", map[string]any{"action": "speed", "speed_ms": 0}); w.Code != 400 {
		t.Fatal("invalid speed")
	}
	if w = call("POST", "/api/runs/"+run.ID+"/control", "a-token", map[string]string{"action": "reset"}); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w = call("GET", "/", "", nil); w.Code != http.StatusOK {
		t.Fatal("no dashboard")
	}
	if b, _ := io.ReadAll(w.Result().Body); !bytes.Contains(b, []byte("Amoeboids")) {
		t.Fatal("dashboard absent")
	}
}
func TestCarryMemoryExclusivity(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db, agents.Registry(), map[string]string{}, web.Handler())
	defer s.Close()
	c := game.DefaultConfig()
	c.CarryMemory = true
	q := NewRequest{Config: c, MemoryNamespace: "learning"}
	if _, err = createSafely(s, "a", q); err != nil {
		t.Fatal(err)
	}
	if _, err = createSafely(s, "a", q); err == nil {
		t.Fatal("concurrent namespace allowed")
	}
	if _, err = createSafely(s, "b", q); err != nil {
		t.Fatal("independent group blocked")
	}
}

func TestCarriedMemorySurvivesApplicationRestart(t *testing.T) {
	path := t.TempDir() + "/memory.db"
	db, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	reg := agents.Registry()
	s := New(db, reg, map[string]string{}, web.Handler())
	c := game.DefaultConfig()
	c.Rounds = 1
	c.CarryMemory = true
	c.Strategies = []string{"cooperative"}
	q := NewRequest{Config: c, MemoryNamespace: "learning"}
	r, err := createSafely(s, "a", q)
	if err != nil {
		t.Fatal(err)
	}
	live := s.live[r.ID]
	for live.record.State.Phase != "complete" {
		advanceSafely(s, live)
	}
	previous := live.engine.Memories()["agent-001"]["decisions"]
	if previous == "" {
		t.Fatal("strategy did not write memory")
	}
	s.Close()
	db.Close()
	db, err = store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s = New(db, reg, map[string]string{}, web.Handler())
	defer s.Close()
	next, err := createSafely(s, "a", q)
	if err != nil {
		t.Fatal(err)
	}
	if s.live[next.ID].engine.Memories()["agent-001"]["decisions"] != previous {
		t.Fatal("memory lost across restart")
	}
	fresh, err := createSafely(s, "b", q)
	if err != nil {
		t.Fatal(err)
	}
	if s.live[fresh.ID].engine.Memories()["agent-001"]["decisions"] != "" {
		t.Fatal("cross-group carry leak")
	}
	c.CarryMemory = false
	clean, err := createSafely(s, "a", NewRequest{Config: c})
	if err != nil {
		t.Fatal(err)
	}
	if s.live[clean.ID].engine.Memories()["agent-001"]["decisions"] != "" {
		t.Fatal("fresh experiment inherited memory")
	}
}

// Direct internal helpers must use the same lock as HTTP handlers and scheduler.
func createSafely(s *Server, owner string, q NewRequest) (store.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.create(owner, q)
}
func advanceSafely(s *Server, r *liveRun) { s.mu.Lock(); defer s.mu.Unlock(); s.advance(r) }
