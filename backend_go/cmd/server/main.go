package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/footballai"
	fmoeWeights "football_sim/pkg/footballai/weights"
	"football_sim/pkg/growth"
	"football_sim/pkg/persistence"
	"football_sim/pkg/server"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"
)

func isPortInUse(host string, port int) bool {
	timeout := 250 * time.Millisecond
	probeHost := host
	if probeHost == "" || probeHost == "0.0.0.0" || probeHost == "::" {
		probeHost = "127.0.0.1"
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(probeHost, fmt.Sprintf("%d", port)), timeout)
	if err == nil {
		_ = conn.Close()
		return true
	}
	return false
}

func getFreePort(host string, preferred int) int {
	port := preferred
	for isPortInUse(host, port) {
		log.Printf("[Server] Port %d is currently in use. Checking port %d...", port, port+1)
		port++
	}
	return port
}

func resolveFile(paths ...string) string {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	for _, rel := range paths {
		if found := findUp(rel); found != "" {
			return found
		}
	}
	if len(paths) > 0 {
		return paths[0]
	}
	return ""
}

func findUp(rel string) string {
	var starts []string
	if cwd, err := os.Getwd(); err == nil {
		starts = append(starts, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(exe))
	}
	for _, start := range starts {
		dir := start
		for i := 0; i < 8; i++ {
			candidate := filepath.Join(dir, rel)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return ""
}

func stdLogger() *log.Logger { return log.Default() }

func main() {
	hostFlag := flag.String("host", "127.0.0.1", "Host/IP to bind (use 0.0.0.0 explicitly for LAN access)")
	portFlag := flag.Int("port", 8000, "Port to bind the HTTP and WebSocket server")
	datasetFlag := flag.String("dataset", "", "Path to dataset.json")
	saveFlag := flag.String("save", "", "Path to career.json save file")
	staticFlag := flag.String("static", "", "Path to compiled frontend dist directory")
	fmoeFlag := flag.String("fmoe", "", "Optional FootballMoE model (.fmoe). Empty keeps the AI-free simulation")
	fmoeFeatures := flag.String("fmoe-features", "", "Comma-separated FootballMoE features to enable: match,injury,rotation,valuation (requires -fmoe)")
	fmoeRecord := flag.Bool("fmoe-record", false, "Record simulation outcomes for future offline training (requires -fmoe)")
	fmoeRecordPath := flag.String("fmoe-record-path", "saves/fmoe_outcomes.jsonl", "Outcome recorder output path")
	flag.Parse()

	// 1. Resolve asset and save paths
	datasetPath := *datasetFlag
	if datasetPath == "" {
		datasetPath = resolveFile("dataset.json", "../dataset.json", "../../dataset.json")
	}

	savePath := *saveFlag
	if savePath == "" {
		savePath = persistence.SavePath()
	}

	staticDir := *staticFlag
	if staticDir == "" {
		staticDir = resolveFile("frontend/dist", "../frontend/dist", "../../frontend/dist")
	}

	log.Printf("[Server] European career server (Go)")
	log.Printf("[Server] Dataset path: %s", datasetPath)
	log.Printf("[Server] Save path: %s", savePath)
	log.Printf("[Server] Static directory: %s", staticDir)
	if info, err := os.Stat(staticDir); err != nil || !info.IsDir() {
		log.Printf("[Server] No compiled frontend at %s. Build it with: cd frontend && bun run build", staticDir)
		log.Printf("[Server] Or run the Vite client: cd frontend && bun run dev (proxies /api and /ws to this port)")
	}

	// 2. Inspect any existing save BEFORE the universe seed is established.
	// Twelve-club Super League saves are no longer playable: they are deleted
	// (save plus seed sidecar) so an ignored legacy career can neither pin a
	// stale seed nor be restored later in boot.
	var snapshot *persistence.CareerSnapshot
	if _, err := os.Stat(savePath); err == nil {
		log.Printf("[Server] Found existing career save at %s. Restoring...", savePath)
		snapshot, err = persistence.LoadCareer(savePath)
		if err != nil {
			log.Fatalf("[Server] FATAL: Could not load existing career save: %v", err)
		}
		if snapshot != nil && snapshot.World == nil {
			log.Printf("[Server] Ignoring 12-club Super League save; starting a fresh European world.")
			snapshot = nil
			_ = persistence.DeleteCareer(savePath)
		}
	} else if !os.IsNotExist(err) {
		log.Fatalf("[Server] FATAL: Could not inspect career save path %s: %v", savePath, err)
	}

	// 3. Initialize simulation systems from one persistent universe seed. Each
	// major subsystem gets an independently-derived deterministic stream.
	// A legacy ignore above leaves no save behind, so the fresh world draws
	// a fresh random seed; continuing world saves reuse their pinned seed.
	universeSeed, err := persistence.LoadOrCreateUniverseSeed(savePath)
	if err != nil {
		log.Fatalf("[Server] FATAL: Could not establish universe seed: %v", err)
	}
	rng := tournament.NewSubsystemRNG(universeSeed)
	ge := growth.NewGrowthEngine(rng.SeedFor("development"))
	dm := datamanager.NewDataManager(datasetPath, ge)
	dm.SetSeed(rng.SeedFor("datamanager"))
	if len(dm.Clubs) == 0 {
		log.Fatalf("[Server] FATAL: Failed to load clubs from dataset at %s", datasetPath)
	}

	// 4. Validate a surviving world save (legacy saves are already gone).
	if snapshot != nil {
		if err := persistence.ValidateCareerSnapshot(snapshot); err != nil {
			log.Fatalf("[Server] FATAL: Existing career save failed validation: %v", err)
		}
	}

	// 5. Build the matching universe before any restore.
	clubs := dm.ClubsList
	if len(clubs) < 2 {
		log.Fatalf("[Server] FATAL: Dataset does not contain enough clubs to initialize a career")
	}
	tm := tournament.NewEuropeanWorldManager(clubs, ge, rng.SeedFor("matches"))
	te := transfers.NewTransferEngine(clubs, tm.Managers, rng.SeedFor("transfers"))
	tm.TransferEngine = te

	// 6. Restore a world save after constructing the matching universe.
	if snapshot != nil {
		loadedVersion := snapshot.Version
		if err := persistence.RestoreCareer(tm, ge, te, snapshot); err != nil {
			log.Fatalf("[Server] FATAL: Could not restore existing career save: %v", err)
		}
		if len(snapshot.ProdigyHomes) > 0 {
			dm.ProdigyHomes = snapshot.ProdigyHomes
		}
		wrote, err := persistence.MaybeWriteMigratedCareer(tm, ge, te, savePath, loadedVersion)
		if err != nil {
			log.Fatalf("[Server] FATAL: Migrated career could not be written; original save preserved: %v", err)
		}
		if wrote {
			log.Printf("[Server] Wrote upgraded career save (version %d)", persistence.SaveVersion)
		}
		log.Printf("[Server] Successfully restored career: Season %s, Matchweek %d", tm.SeasonName, tm.CurrentMatchweek)
	} else {
		log.Printf("[Server] No prior career save found. Initialized fresh Top Five European universe.")
	}

	if err := tm.ValidateWorldState(); err != nil {
		log.Fatalf("[Server] FATAL: Universe failed startup validation: %v", err)
	}

	// 7. Load the optional FootballMoE brain. All feature flags default to
	// off; a career whose save pins a different model hash keeps its
	// behavior — mismatches disable AI features for the session with a log
	// line, never a silent behavior change.
	aiCfg := footballai.AIConfig{ModelPath: *fmoeFlag}
	if *fmoeFlag != "" {
		// Verify checksum and architecture before the universe boots so a
		// corrupt model is a hard, early failure — never a mid-career one.
		if err := fmoeWeights.Verify(*fmoeFlag); err != nil {
			log.Fatalf("[Server] FATAL: FootballMoE model failed verification: %v", err)
		}
		for _, f := range strings.Split(*fmoeFeatures, ",") {
			switch strings.TrimSpace(f) {
			case "match":
				aiCfg.UseMatchModel = true
			case "injury":
				aiCfg.UseInjuryModel = true
			case "rotation":
				aiCfg.UseRotationModel = true
			case "valuation":
				aiCfg.UseValuationModel = true
			}
		}
		if *fmoeRecord {
			aiCfg.RecordOutcomes = true
			aiCfg.RecordPath = resolveFile(*fmoeRecordPath, "../"+*fmoeRecordPath)
		}
	}
	brain, err := footballai.NewBrain(aiCfg, stdLogger())
	if err != nil {
		log.Fatalf("[Server] FATAL: FootballMoE model failed to load: %v", err)
	}
	if brain.Enabled() && snapshot != nil && snapshot.AIModel != nil {
		if !snapshot.AIModel.Matches(brain.Info()) {
			log.Printf("[Server] Career pins a different FootballMoE model (save hash %s, loaded hash %s).", snapshot.AIModel.ModelHash, brain.Info().ModelHash)
			log.Printf("[Server] AI features stay disabled for this session; the career keeps its deterministic behavior.")
			brain.DisableFeatures()
		}
	}

	// 7a. Hand the brain to the world and market engines. Feature flags stay
	// default-off: nil brain or disabled flags leave both engines on their
	// existing deterministic paths.
	tm.AIBrain = brain
	te.AIBrain = brain

	// 7b. Configure the simulation-only HTTP server.
	port := getFreePort(*hostFlag, *portFlag)
	srv := server.NewServer(dm, ge, tm, te, savePath, staticDir)
	srv.SetAIBrain(brain)

	httpServer := &http.Server{
		Addr:              net.JoinHostPort(*hostFlag, fmt.Sprintf("%d", port)),
		Handler:           srv,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 8. Graceful shutdown handler
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		displayHost := *hostFlag
		if displayHost == "127.0.0.1" || displayHost == "::1" {
			displayHost = "localhost"
		}
		log.Printf("[Server] 🚀 Server running at http://%s:%d", displayHost, port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Server] ListenAndServe error: %v", err)
		}
	}()

	<-stopCh
	log.Printf("[Server] Shutting down gracefully...")

	srv.Stop()
	brain.Close()

	// Persist state on shutdown. The universe seed remains in its sidecar for
	// this career and is restored before any simulation system is initialized.
	if saved, err := persistence.SaveCareer(tm, ge, te, savePath); err == nil {
		log.Printf("[Server] Saved latest career snapshot to %s", saved)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("[Server] HTTP shutdown error: %v", err)
	}

	log.Printf("[Server] Server stopped successfully.")
}
