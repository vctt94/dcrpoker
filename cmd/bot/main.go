package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/decred/dcrd/dcrutil/v4"
	_ "github.com/mattn/go-sqlite3" // SQLite3 driver
	kit "github.com/vctt94/bisonbotkit"
	"github.com/vctt94/pokerbisonrelay/pkg/bot"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/server"
)

var (
	dataDir            = flag.String("datadir", "", "Data directory for bot files")
	url                = flag.String("url", "", "Server URL")
	grpcServerCertPath = flag.String("grpcservercert", "", "Path to gRPC server certificate")
	certFile           = flag.String("cert", "", "Path to certificate file")
	keyFile            = flag.String("key", "", "Path to key file")
	rpcUser            = flag.String("rpcuser", "", "RPC username")
	rpcPass            = flag.String("rpcpass", "", "RPC password")
	grpcHost           = flag.String("grpchost", "", "gRPC host address")
	grpcPort           = flag.String("grpcport", "", "gRPC port")
	debugLevel         = flag.String("debuglevel", "", "Debug level")
	metricsAddr        = flag.String("metricsaddr", "127.0.0.1:9090", "Address to serve Prometheus metrics (host:port). Empty to disable.")
)

func realMain() error {
	// Parse flags
	flag.Parse()

	// Load configuration
	cfg, err := bot.LoadBotConfig("pokerbot", *dataDir)
	if err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	// Override config with flags if provided
	if *grpcHost != "" {
		cfg.Config.ExtraConfig["grpchost"] = *grpcHost
	}
	if *grpcPort != "" {
		cfg.Config.ExtraConfig["grpcport"] = *grpcPort
	}
	if *certFile != "" {
		cfg.CertFile = *certFile
	}
	if *keyFile != "" {
		cfg.KeyFile = *keyFile
	}

	// Rebuild server address if gRPC host/port were overridden
	if *grpcHost != "" || *grpcPort != "" {
		grpcHostVal := cfg.Config.ExtraConfig["grpchost"]
		grpcPortVal := cfg.Config.ExtraConfig["grpcport"]
		if grpcHostVal == "" {
			return fmt.Errorf("GRPCHost is required")
		}
		if grpcPortVal == "" {
			return fmt.Errorf("GRPCPort is required")
		}
		cfg.ServerAddress = fmt.Sprintf("%s:%s", grpcHostVal, grpcPortVal)
	}

	// Create channels for handling PMs and tips
	pmChan := make(chan *types.ReceivedPM)
	tipChan := make(chan *types.ReceivedTip)
	tipProgressChan := make(chan *types.TipProgressEvent)

	cfg.Config.PMChan = pmChan
	cfg.Config.TipProgressChan = tipProgressChan
	cfg.Config.TipReceivedChan = tipChan

	botInstance, err := kit.NewBot(cfg.Config)
	if err != nil {
		return fmt.Errorf("failed to create bot: %v", err)
	}
	log := botInstance.LogBackend.Logger("BOT")

	log.Infof("Starting bot...")
	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer botInstance.Close()

	// Initialize database
	db, err := server.NewDatabase(filepath.Join(cfg.DataDir, "poker.db"))
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize and start the gRPC poker server
	grpcServer, grpcLis, pokerServer, err := bot.SetupGRPCServer(cfg.DataDir, cfg.CertFile, cfg.KeyFile, cfg.ServerAddress, db, botInstance.LogBackend)
	if err != nil {
		return fmt.Errorf("failed to setup gRPC server: %v", err)
	}

	// Initialize bot state
	state := bot.NewState(db)
	// Optional metrics
	if *metricsAddr != "" {
		go func() {
			mux := http.NewServeMux()
			// Register simple HTML status UI and metrics endpoint
			parseTemplatesOnce()

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/status", http.StatusFound)
			})
			mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
				tables := pokerServer.GetAllTables()
				drops := poker.GetEventMetricsSnapshot().Dropped + server.GetMetrics().EventDropsTotal()
				data := statusPageData{
					GeneratedAt:   time.Now(),
					QueueDepth:    pokerServer.EventQueueDepth(),
					QueueCapacity: pokerServer.EventQueueCapacity(),
					EventDrops:    drops,
					GoRoutines:    runtime.NumGoroutine(),
					Summary:       Stats(tables),
				}
				if err := statusTmpl.Execute(w, data); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			})
			mux.HandleFunc("/metrics", handleMetrics(pokerServer))
			srv := &http.Server{Addr: *metricsAddr, Handler: mux}
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Errorf("metrics serve error: %v", err)
			}
		}()
		log.Infof("Metrics endpoint listening on http://%s/metrics", *metricsAddr)
	}

	// Start gRPC server
	errCh := make(chan error, 1)
	go func() {
		log.Infof("Starting gRPC poker server on %s", cfg.ServerAddress)
		errCh <- grpcServer.Serve(grpcLis)
	}()
	defer grpcServer.Stop() // Ensure gRPC server is stopped on exit
	// Handle PMs
	go func() {
		for pm := range pmChan {
			state.HandlePM(ctx, botInstance, pm)
		}
	}()

	// Handle received tips
	go func() {
		for tip := range tipChan {
			var userID zkidentity.ShortID
			userID.FromBytes(tip.Uid)

			log.Infof("Tip received: %.8f DCR from %s",
				dcrutil.Amount(tip.AmountMatoms/1e3).ToCoin(),
				userID.String())

			// Update player balance
			err := db.UpdatePlayerBalance(ctx, userID.String(), int64(tip.AmountMatoms/1e3),
				"tip", "Received tip from user")
			if err != nil {
				log.Errorf("Failed to update player balance: %v", err)
				botInstance.SendPM(ctx, userID.String(),
					"Error updating your balance. Please contact support.")
			} else {
				botInstance.SendPM(ctx, userID.String(),
					fmt.Sprintf("Thank you for the tip of %.8f DCR! Your balance has been updated.",
						dcrutil.Amount(tip.AmountMatoms/1e3).ToCoin()))
			}

			botInstance.AckTipReceived(ctx, tip.SequenceId)
		}
	}()

	// Handle tip progress updates
	go func() {
		for progress := range tipProgressChan {
			log.Infof("Tip progress event (sequence ID: %d)", progress.SequenceId)
			err := botInstance.AckTipProgress(ctx, progress.SequenceId)
			if err != nil {
				log.Errorf("Failed to acknowledge tip progress: %v", err)
			}
		}
	}()

	// Signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Run the bot concurrently so we can observe signals and server errors
	runCh := make(chan error, 1)
	go func() { runCh <- botInstance.Run(ctx) }()

	select {
	case <-sigCh:
		// Graceful stop: stop poker internals then gRPC
		pokerServer.Stop()
		done := make(chan struct{})
		go func() { grpcServer.GracefulStop(); close(done) }()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			grpcServer.Stop()
		}
		cancel()
		log.Infof("Shutdown complete")
		return nil
	case err := <-errCh:
		if err != nil {
			log.Errorf("gRPC server error: %v", err)
		}
		cancel()
		return err
	case err := <-runCh:
		// Bot exited on its own; stop gRPC as well
		pokerServer.Stop()
		grpcServer.Stop()
		return err
	}
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
