package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite3 driver
	"github.com/vctt94/bisonbotkit/utils"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/server"
)

const appName = "pokerbot"

var (
	datadirFlag        = flag.String("datadir", "", "Data directory for bot files")
	grpcServerCertPath = flag.String("grpcservercert", "", "Path to gRPC server certificate")
	grpcHost           = flag.String("grpchost", "", "gRPC host address")
	grpcPort           = flag.String("grpcport", "", "gRPC port")
	debug              = flag.String("debug", "info", "Debug level")
	metricsAddr        = flag.String("metricsaddr", "0.0.0.0:9091", "Address to serve Prometheus metrics (host:port). Empty to disable.")
)

func realMain() error {
	// Parse flags
	flag.Parse()

	datadir := utils.AppDataDir(appName, false)
	if datadir == "" {
		if *datadirFlag != "" {
			datadir = *datadirFlag
		} else {
			return fmt.Errorf("data directory is required")
		}
	}
	// Load configuration
	cfg, err := server.LoadServerConfig(datadir, appName+".conf")
	if err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	// Override config with flags if provided
	if *datadirFlag != "" {
		cfg.Datadir = *datadirFlag
	}
	if *grpcHost != "" {
		cfg.GRPCHost = *grpcHost
	}
	if *grpcPort != "" {
		cfg.GRPCPort = *grpcPort
	}
	if *grpcServerCertPath != "" {
		cfg.GRPCCertPath = *grpcServerCertPath
	}

	log := cfg.LogBackend.Logger(appName)

	pokerServer, err := server.NewServer(*cfg)
	if err != nil {
		return fmt.Errorf("failed to create poker server: %v", err)
	}

	// Optional metrics
	if *metricsAddr != "" {
		go func() {
			mux := http.NewServeMux()
			// Register simple HTML status UI and metrics endpoint
			parseTemplatesOnce()

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/metrics", http.StatusFound)
			})
			// Prometheus scraping endpoint
			mux.HandleFunc("/metrics/prometheus", func(w http.ResponseWriter, r *http.Request) {
				sstats := Stats(pokerServer)
				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprint(w, formatPrometheusMetrics(pokerServer, sstats))
			})

			mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
				drops := poker.GetEventMetricsSnapshot().Dropped + server.GetMetrics().EventDropsTotal()
				// Get online users (notification streams) and categorized in-game users for debugging
				onlineUsers := pokerServer.GetAllOnlineUsers()
				_, inGameUsers := pokerServer.GetInLobbyAndInGameUsers()
				// Debug: log what we're seeing
				log.Debugf("Metrics collection: onlineUsers=%d, inGameUsers=%d", len(onlineUsers), len(inGameUsers))
				if len(onlineUsers) == 0 && len(inGameUsers) > 0 {
					log.Warnf("WARNING: Found %d in-game users but 0 online users - possible notification stream missing!", len(inGameUsers))
					// Log which players are in-game but not online
					for playerID := range inGameUsers {
						if !onlineUsers[playerID] {
							log.Warnf("Player %s has game stream but NO notification stream", playerID)
						}
					}
				}
				sstats := Stats(pokerServer)

				// Serve HTML status page
				data := statusPageData{
					GeneratedAt:       time.Now(),
					QueueDepth:        pokerServer.EventQueueDepth(),
					QueueCapacity:     pokerServer.EventQueueCapacity(),
					EventDrops:        drops,
					GoRoutines:        runtime.NumGoroutine(),
					Summary:           sstats,
					PrometheusMetrics: formatPrometheusMetrics(pokerServer, sstats),
					Metrics:           buildMetricsData(pokerServer, sstats),
				}
				if err := statusTmpl.Execute(w, data); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			})
			srv := &http.Server{Addr: *metricsAddr, Handler: mux}
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Errorf("metrics serve error: %v", err)
			}
		}()
		log.Infof("Metrics endpoint listening on http://%s/metrics", *metricsAddr)
	}

	// Signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh
	log.Infof("Shutting down...")
	pokerServer.Stop()
	log.Infof("Shutdown complete")
	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
