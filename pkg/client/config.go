package client

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Default gRPC connection settings for production deployment
const (
	defaultGRPCHost = "178.156.178.191"
	defaultGRPCPort = "50050"
)

// defaultServerCertPEM is written to <datadir>/ca.cert on first run when creating
// a default config, so the UI has a usable TLS cert path out of the box.
const defaultServerCertPEM = `-----BEGIN CERTIFICATE-----
MIIBiTCCAS+gAwIBAgIRAMS59RdukJ0c41QV6H1KdoUwCgYIKoZIzj0EAwIwJDEQ
MA4GA1UEChMHUG9rZXJDQTEQMA4GA1UEAxMHUG9rZXJDQTAeFw0yNTExMjExNzUw
NTZaFw0zNTExMjAxNzUwNTZaMCQxEDAOBgNVBAoTB1Bva2VyQ0ExEDAOBgNVBAMT
B1Bva2VyQ0EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAR2Ac7OAXooOpL3GezW
eJsGnGjqBlmIXrajh+c7cmF2cW5HsiPqeMdb8o0FQ6z7QBLIJJAsTax1Gwbqdjv1
QYopo0IwQDAOBgNVHQ8BAf8EBAMCAoQwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4E
FgQU3RgmPKabIQLOqV2UPbS+7pNTH2YwCgYIKoZIzj0EAwIDSAAwRQIhAJRVw7od
EZgyT95h5FRiDyhEmgH/1HsZUnuiwjHPxNWKAiAb6fGY30A/HVX6jPWpzC/hNJfI
WXPGJtBI+49aqzOyJQ==
-----END CERTIFICATE-----`

// PokerConf is the config loaded from our .conf
type PokerConf struct {
	// Absolute directory where the config/logs live.
	Datadir string

	// Extracted Poker gRPC settings
	GRPCHost     string // gRPC server hostname
	GRPCPort     string // gRPC server port
	GRPCCertPath string // Path to gRPC server certificate

	PayoutAddress string

	LogFile        string
	Debug          string
	MaxLogFiles    int
	MaxBufferLines int
}

// parseClientConfigFile parses the config file at the given path into a PokerConf struct.
func parseClientConfigFile(configPath string, appName string) (*PokerConf, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &PokerConf{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "datadir":
			cfg.Datadir = value
		case "grpchost":
			cfg.GRPCHost = value
		case "grpcport":
			cfg.GRPCPort = value
		case "grpccertpath":
			cfg.GRPCCertPath = value
		case "payoutaddress":
			cfg.PayoutAddress = value
		case "logfile":
			cfg.LogFile = value
			if cfg.LogFile == "" {
				cfg.LogFile = filepath.Join(cfg.Datadir, "logs", appName+".log")
			}
		case "debug":
			cfg.Debug = value
			if cfg.Debug == "" {
				cfg.Debug = "info"
			}
		case "maxlogfiles":
			fmt.Sscanf(value, "%d", &cfg.MaxLogFiles)
			if cfg.MaxLogFiles == 0 {
				cfg.MaxLogFiles = 5
			}
		case "maxbufferlines":
			fmt.Sscanf(value, "%d", &cfg.MaxBufferLines)
			if cfg.MaxBufferLines == 0 {
				cfg.MaxBufferLines = 1000
			}
		default:
			// Ignore unknown keys to preserve forward-compatibility with older configs.
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Apply defaults for missing values (only datadir is truly required)
	if strings.TrimSpace(cfg.Datadir) == "" {
		return nil, fmt.Errorf("missing required field: datadir")
	}

	// Apply defaults for gRPC settings if not specified
	if strings.TrimSpace(cfg.GRPCHost) == "" {
		cfg.GRPCHost = defaultGRPCHost
	}
	if strings.TrimSpace(cfg.GRPCPort) == "" {
		cfg.GRPCPort = defaultGRPCPort
	}
	if strings.TrimSpace(cfg.GRPCCertPath) == "" {
		cfg.GRPCCertPath = filepath.Join(cfg.Datadir, "ca.cert")
	}

	// Apply defaults for other optional fields
	if cfg.LogFile == "" {
		cfg.LogFile = filepath.Join(cfg.Datadir, "logs", appName+".log")
	}
	if cfg.Debug == "" {
		cfg.Debug = "info"
	}
	if cfg.MaxLogFiles == 0 {
		cfg.MaxLogFiles = 5
	}
	if cfg.MaxBufferLines == 0 {
		cfg.MaxBufferLines = 1000
	}

	return cfg, nil
}

// LoadClientConf attempts to load the client config (.conf) from the default locations.
func LoadClientConf(configPath string, fileName string) (*PokerConf, error) {
	// Check if fileName has .conf extension
	if !strings.HasSuffix(fileName, ".conf") {
		return nil, fmt.Errorf("filename must have .conf extension, got: %s", fileName)
	}

	// Get app name by removing .conf extension
	appName := strings.TrimSuffix(fileName, ".conf")

	// Require explicit configPath; callers must provide the sandboxed dir.
	if strings.TrimSpace(configPath) == "" {
		return nil, fmt.Errorf("configPath is required")
	}

	// Ensure the config directory exists
	if err := os.MkdirAll(configPath, 0700); err != nil {
		return nil, err
	}

	// Try to load existing config
	fullPath := filepath.Join(configPath, fileName)
	if _, err := os.Stat(fullPath); err == nil {
		cfg, err := parseClientConfigFile(fullPath, appName)
		if err != nil {
			return nil, err
		}
		// Ensure default certificate exists if cert path is set
		if cfg.GRPCCertPath != "" {
			if _, err := os.Stat(cfg.GRPCCertPath); os.IsNotExist(err) {
				if err := CreateDefaultServerCert(cfg.GRPCCertPath); err != nil {
					return nil, fmt.Errorf("failed to create default server cert: %w", err)
				}
			}
		}
		return cfg, nil
	}

	// Create default config with production deployment defaults.
	// The Flutter UI and CLI can both override these values later, but this
	// keeps first-run working without requiring user input.
	cfg := &PokerConf{
		Datadir:        configPath,
		GRPCCertPath:   filepath.Join(configPath, "ca.cert"),
		GRPCHost:       defaultGRPCHost,
		GRPCPort:       defaultGRPCPort,
		LogFile:        filepath.Join(configPath, "logs", appName+".log"),
		Debug:          "info",
		MaxLogFiles:    5,
		MaxBufferLines: 1000,
	}

	// Write default config
	if err := writeClientConfigFile(cfg, fullPath); err != nil {
		return nil, err
	}

	// Write default certificate
	if err := CreateDefaultServerCert(cfg.GRPCCertPath); err != nil {
		return nil, fmt.Errorf("failed to create default server cert: %w", err)
	}

	return cfg, nil
}

// WriteClientConfigFile writes the configuration to a file.
func WriteClientConfigFile(cfg *PokerConf, configPath string) error {
	configData := fmt.Sprintf(
		`datadir=%s
grpchost=%s
grpcport=%s
grpccertpath=%s
payoutaddress=%s
logfile=%s
debug=%s
maxlogfiles=%d
maxbufferlines=%d
`,
		cfg.Datadir,
		cfg.GRPCHost,
		cfg.GRPCPort,
		cfg.GRPCCertPath,
		cfg.PayoutAddress,
		cfg.LogFile,
		cfg.Debug,
		cfg.MaxLogFiles,
		cfg.MaxBufferLines,
	)

	return os.WriteFile(configPath, []byte(configData), 0600)
}

// writeClientConfigFile is a wrapper for backward compatibility.
func writeClientConfigFile(cfg *PokerConf, configPath string) error {
	return WriteClientConfigFile(cfg, configPath)
}

// CreateDefaultServerCert creates a basic server certificate file for testing
// Note: In production, you should use a proper certificate from your server
func CreateDefaultServerCert(certPath string) error {
	// Create directory for cert file if it doesn't exist
	dir := filepath.Dir(certPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create cert directory %s: %v", dir, err)
	}

	// Write the certificate file
	if err := os.WriteFile(certPath, []byte(defaultServerCertPEM), 0600); err != nil {
		return fmt.Errorf("failed to write cert file %s: %v", certPath, err)
	}

	return nil
}

// SetupGRPCConnection sets up a GRPC connection with TLS credentials
func SetupGRPCConnection(serverAddr, certPath, grpcHost string) (*grpc.ClientConn, error) {
	// Load the server certificate
	pemServerCA, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read server certificate: %v", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server certificate to pool")
	}

	// Create the TLS credentials with ServerName set to grpcHost
	tlsConfig := &tls.Config{
		RootCAs:    certPool,
		ServerName: grpcHost,
	}

	creds := credentials.NewTLS(tlsConfig)
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}

	// Create the client connection
	conn, err := grpc.Dial(serverAddr, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	return conn, nil
}
