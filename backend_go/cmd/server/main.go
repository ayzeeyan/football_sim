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
	"syscall"
	"time"

	"football_sim/pkg/datamanager"
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

func main() {
	hostFlag := flag.String("host", "127.0.0.1", "Host/IP to bind (use 0.0.0.0 explicitly for LAN access)")
	portFlag := flag.Int("port", 8000, "Port to bind the HTTP and WebSocket server")
	datasetFlag := flag.String("dataset", "", "Path to dataset.json")
	saveFlag := flag.String("save", "", "Path to career.json save file")
	staticFlag := flag.String("static", "", "Path to compiled frontend dist directory")
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

	log.Printf("[Server] Super League career server (Go)")
	log.Printf("[Server] Dataset path: %s", datasetPath)
	log.Printf("[Server] Save path: %s", savePath)
	log.Printf("[Server] Static directory: %s", staticDir)
	if info, err := os.Stat(staticDir); err != nil || !info.IsDir() {
		log.Printf("[Server] No compiled frontend at %s. Build it with: cd frontend && bun run build", staticDir)
		log.Printf("[Server] Or run the Vite client: cd frontend && bun run dev (proxies /api and /ws to this port)")
	}

	// 2. Initialize simulation systems
	ge := growth.NewGrowthEngine(time.Now().UnixNano())
	dm := datamanager.NewDataManager(datasetPath, ge)
	if len(dm.Clubs) == 0 {
		log.Fatalf("[Server] FATAL: Failed to load clubs from dataset at %s", datasetPath)
	}

	eliteClubs := dm.GetEliteClubs()
	if len(eliteClubs) != 12 {
		log.Fatalf("[Server] FATAL: Expected 12 elite clubs, found %d", len(eliteClubs))
	}

	tm := tournament.NewTournamentManager(eliteClubs, ge, time.Now().UnixNano())
	te := transfers.NewTransferEngine(eliteClubs, tm.Managers, time.Now().UnixNano())
	tm.TransferEngine = te

	// 3. Attempt restoring previous career snapshot. Existing saves are never
	// silently discarded: malformed/corrupt state is a startup error so the
	// user can diagnose or recover the save instead of unknowingly replacing it.
	if _, err := os.Stat(savePath); err == nil {
		log.Printf("[Server] Found existing career save at %s. Restoring...", savePath)
		snap, err := persistence.LoadCareer(savePath)
		if err != nil {
			log.Fatalf("[Server] FATAL: Could not load existing career save: %v", err)
		}
		if err := persistence.ValidateCareerSnapshot(snap); err != nil {
			log.Fatalf("[Server] FATAL: Existing career save failed validation: %v", err)
		}
		if err := persistence.RestoreCareer(tm, ge, te, snap); err != nil {
			log.Fatalf("[Server] FATAL: Could not restore existing career save: %v", err)
		}
		if err := tm.ValidateWorldState(); err != nil {
			log.Fatalf("[Server] FATAL: Restored career failed world validation: %v", err)
		}
		if len(snap.ProdigyHomes) > 0 {
			dm.ProdigyHomes = snap.ProdigyHomes
		}
		log.Printf("[Server] Successfully restored career: Season %s, Matchweek %d", tm.SeasonName, tm.CurrentMatchweek)
	} else if !os.IsNotExist(err) {
		log.Fatalf("[Server] FATAL: Could not inspect career save path %s: %v", savePath, err)
	} else {
		log.Printf("[Server] No prior career save found. Initialized fresh Super League universe.")
	}

	if err := tm.ValidateWorldState(); err != nil {
		log.Fatalf("[Server] FATAL: Universe failed startup validation: %v", err)
	}

	// 4. Configure HTTP & WebSocket Server
	port := getFreePort(*hostFlag, *portFlag)
	srv := server.NewServer(dm, ge, tm, te, savePath, staticDir)

	httpServer := &http.Server{
		Addr:              net.JoinHostPort(*hostFlag, fmt.Sprintf("%d", port)),
		Handler:           srv,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 5. Graceful shutdown handler
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

	// Persist state on shutdown
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
