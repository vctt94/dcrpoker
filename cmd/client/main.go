package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/vctt94/pokerbisonrelay/pkg/client"
	"github.com/vctt94/pokerbisonrelay/pkg/ui"
)

var (
	// Define command line flags
	dataDir         = flag.String("datadir", "", "Directory to load config file from")
	payoutAddress   = flag.String("payoutaddress", "", "Address to payout to")
	rpcURL          = flag.String("url", "", "URL of the websocket endpoint")
	grpcServerCert  = flag.String("grpcservercert", "", "Path to server.crt file for TLS")
	brClientCert    = flag.String("brclientcert", "", "Path to brclient rpc.cert file")
	brClientRPCCert = flag.String("brclientrpc.cert", "", "Path to rpc-client.cert file")
	brClientRPCKey  = flag.String("brclientrpc.key", "", "Path to rpc-client.key file")
	rpcUser         = flag.String("rpcuser", "", "RPC user for basic authentication")
	rpcPass         = flag.String("rpcpass", "", "RPC password for basic authentication")
	grpcHost        = flag.String("grpchost", "", "GRPC server hostname")
	grpcPort        = flag.String("grpcport", "", "GRPC server port")
	logFile         = flag.String("logfile", "", "Path to log file")
	maxLogFiles     = flag.Int("maxlogfiles", 10, "Maximum number of log files")
	maxBufferLines  = flag.Int("maxbufferlines", 1000, "Maximum number of buffer lines")
	debug           = flag.String("debug", "", "Debug level for logging")
)

func main() {
	flag.Parse()

	cfg, err := client.LoadClientConfig(*dataDir, "pokerclient.conf")
	if err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create poker client with configuration
	pokerClient, err := client.NewPokerClient(ctx, cfg)
	if err != nil {
		fmt.Printf("Failed to create poker client: %v\n", err)
		os.Exit(1)
	}
	defer pokerClient.Close()
	log := cfg.LogBackend.Logger("pokerclient")

	// Start the notification stream
	if err := pokerClient.StartNotificationStream(ctx); err != nil {
		log.Infof("Failed to start notifications: %v\n", err)
		os.Exit(1)
	}

	// Restore game stream if player is already at a table (similar to Flutter UI)
	// This ensures metrics correctly count online users when TUI client reopens
	if tableID, err := pokerClient.GetPlayerCurrentTable(ctx); err == nil && tableID != "" {
		pokerClient.SetCurrentTableID(tableID)
		if err := pokerClient.StartGameStream(ctx); err != nil {
			log.Infof("Failed to restore game stream for existing table %s: %v", tableID, err)
			// Don't exit - allow client to continue without game stream
		} else {
			log.Infof("Restored game stream for table %s", tableID)
		}
	}

	// Start the UI with the client's components
	ui.Run(ctx, pokerClient)
}
