package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"somas/internal/api"
	"somas/internal/store"
	"somas/pkg/agents"
	"somas/pkg/game"
	"somas/web"
	"strings"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	dbpath := flag.String("db", "data/somas.db", "SQLite file")
	batch := flag.Bool("batch", false, "run headless experiment")
	cfgpath := flag.String("config", "", "JSON config file (partial overrides)")
	seed := flag.Int64("seed", 42, "seed")
	strategy := flag.String("strategies", "random", "comma-separated repeating strategy mix")
	rounds := flag.Int("rounds", 30, "rounds per cycle")
	repeat := flag.Int("repeat", 1, "batch trials, incrementing seed")
	out := flag.String("out", "experiment.json", "batch JSON output")
	flag.Parse()
	registry := agents.Registry() // Register approved custom agents here; see AGENT_DEVELOPMENT.md.
	c := game.DefaultConfig()
	c.Seed = *seed
	c.Strategies = strings.Split(*strategy, ",")
	c.Rounds = *rounds
	if *cfgpath != "" {
		f, err := os.Open(*cfgpath)
		if err != nil {
			return err
		}
		defer f.Close()
		d := json.NewDecoder(f)
		d.DisallowUnknownFields()
		if err = d.Decode(&c); err != nil {
			return err
		}
	}
	if *batch {
		if *repeat < 1 || *repeat > 1000 {
			return fmt.Errorf("repeat must be 1–1000")
		}
		type trial struct {
			Config  game.Config     `json:"config"`
			History []game.Snapshot `json:"history"`
		}
		trials := []trial{}
		f, err := os.Create(*out + ".csv")
		if err != nil {
			return err
		}
		defer f.Close()
		cw := csv.NewWriter(f)
		_ = cw.Write([]string{"seed", "round", "utility", "cpr", "alive", "ctk", "cooperation", "violations"})
		baseSeed := c.Seed
		for i := 0; i < *repeat; i++ {
			c.Seed = baseSeed + int64(i)
			e, err := game.New(c, registry, nil)
			if err != nil {
				return err
			}
			t := trial{Config: c, History: []game.Snapshot{e.Snapshot()}}
			for e.Step() {
				s := e.Snapshot()
				t.History = append(t.History, s)
				if s.Phase == "round_end" {
					_ = cw.Write([]string{fmt.Sprint(c.Seed), fmt.Sprint(s.Round), fmt.Sprint(s.Metrics.Utility), fmt.Sprint(s.Metrics.CPR), fmt.Sprint(s.Metrics.Alive), fmt.Sprint(s.Metrics.CTK), fmt.Sprint(s.Metrics.Cooperation), fmt.Sprint(s.Metrics.Violations)})
				}
			}
			trials = append(trials, t)
		}
		cw.Flush()
		if err = cw.Error(); err != nil {
			return err
		}
		b, err := json.MarshalIndent(trials, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(*out, b, 0600)
	}
	keys := map[string]string{}
	raw := os.Getenv("SOMAS_GROUPS")
	if raw == "" {
		return fmt.Errorf("set SOMAS_GROUPS='group=long-random-token[,other=other-token]' (tokens >=16 characters)")
	}
	used := map[string]bool{}
	for _, entry := range strings.Split(raw, ",") {
		p := strings.SplitN(entry, "=", 2)
		if len(p) != 2 || p[0] == "" || len(p[1]) < 16 || keys[p[0]] != "" || used[p[1]] {
			return fmt.Errorf("invalid or duplicate SOMAS_GROUPS entry")
		}
		keys[p[0]] = p[1]
		used[p[1]] = true
	}
	db, err := store.Open(*dbpath)
	if err != nil {
		return err
	}
	defer db.Close()
	app := api.New(db, registry, keys, web.Handler())
	defer app.Close()
	srv := &http.Server{Addr: *addr, Handler: app, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	log.Printf("SOMAS dashboard at http://%s", *addr)
	err = srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
