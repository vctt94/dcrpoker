package client

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

	// UI-only settings
	SoundsEnabled bool   // Enable/disable sound effects in UI
	TableTheme    string // Visual theme for the poker table (e.g., "decred", "classic")
	CardTheme     string // Visual theme for playing cards (e.g., "standard", "decred")
	HideTableLogo bool   // Whether to hide the center table logo overlay
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
		case "soundsenabled":
			cfg.SoundsEnabled = value == "1" || strings.ToLower(value) == "true"
		case "tabletheme":
			cfg.TableTheme = strings.TrimSpace(strings.ToLower(value))
		case "cardtheme":
			cfg.CardTheme = strings.TrimSpace(strings.ToLower(value))
		case "hidetablelogo":
			cfg.HideTableLogo = value == "1" || strings.ToLower(value) == "true"
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
	if cfg.TableTheme == "" {
		cfg.TableTheme = "classic"
	}
	if cfg.CardTheme == "" {
		cfg.CardTheme = "standard"
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
		SoundsEnabled:  true, // Default to enabled
		TableTheme:     "classic",
		CardTheme:      "standard",
		HideTableLogo:  false,
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
	soundsEnabledVal := 0
	if cfg.SoundsEnabled {
		soundsEnabledVal = 1
	}
	hideLogoVal := 0
	if cfg.HideTableLogo {
		hideLogoVal = 1
	}
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
soundsenabled=%d
tabletheme=%s
cardtheme=%s
hidetablelogo=%d
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
		soundsEnabledVal,
		cfg.TableTheme,
		cfg.CardTheme,
		hideLogoVal,
	)

	return os.WriteFile(configPath, []byte(configData), 0600)
}

// writeClientConfigFile is a wrapper for backward compatibility.
func writeClientConfigFile(cfg *PokerConf, configPath string) error {
	return WriteClientConfigFile(cfg, configPath)
}

// ValidTableThemes lists all valid table theme keys
var ValidTableThemes = map[string]bool{
	"classic":        true,
	"decred":         true,
	"decred_inverse": true,
}

// ValidCardThemes lists all valid card theme keys
var ValidCardThemes = map[string]bool{
	"standard": true,
	"decred":   true,
}

// ValidateCertPath checks if the certificate path is valid
func ValidateCertPath(certPath string) error {
	if certPath == "" {
		return nil // Empty is allowed (will use default)
	}

	// Check if path is absolute or relative
	absPath := certPath
	if !filepath.IsAbs(certPath) {
		// For relative paths, check if the directory exists
		dir := filepath.Dir(certPath)
		if dir != "." && dir != "" {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return fmt.Errorf("certificate directory does not exist: %s", dir)
			}
		}
	} else {
		// For absolute paths, check if parent directory exists
		dir := filepath.Dir(absPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return fmt.Errorf("certificate directory does not exist: %s", dir)
		}
	}

	return nil
}

// XXX Remove address option from config, as it will only be set by sign
// address.
func ValidateAddress(addr string) error {
	if addr == "" {
		return nil // Empty is allowed
	}

	addr = strings.TrimSpace(addr)

	// Check if it's a hex-encoded pubkey (33 or 65 bytes = 66 or 130 hex chars)
	if len(addr) == 66 || len(addr) == 130 {
		// Validate hex format
		decoded, err := hex.DecodeString(addr)
		if err != nil {
			return fmt.Errorf("invalid hex format for pubkey: %v", err)
		}
		// Validate length matches expected pubkey sizes
		if len(decoded) != 33 && len(decoded) != 65 {
			return fmt.Errorf("invalid pubkey length: expected 33 or 65 bytes, got %d", len(decoded))
		}
		return nil
	}

	// If we couldn't decode as address or pubkey, return error
	return fmt.Errorf("invalid address/pubkey format: must be a valid Decred address or hex-encoded 33/65-byte pubkey")
}

// ValidateTheme checks if a theme key is valid
func ValidateTheme(theme string, validThemes map[string]bool, themeType string) error {
	if theme == "" {
		return nil // Empty is allowed (will use default)
	}

	normalized := strings.ToLower(strings.TrimSpace(theme))
	if !validThemes[normalized] {
		validKeys := make([]string, 0, len(validThemes))
		for k := range validThemes {
			validKeys = append(validKeys, k)
		}
		return fmt.Errorf("invalid %s theme '%s' (valid options: %v)", themeType, theme, validKeys)
	}

	return nil
}

// ValidateServerAddress validates server address format (host:port)
func ValidateServerAddress(serverAddr string) error {
	if serverAddr == "" {
		return nil // Empty is allowed
	}

	host, port, ok := strings.Cut(serverAddr, ":")
	if !ok || host == "" || port == "" {
		return fmt.Errorf("invalid server address format (expected host:port): %s", serverAddr)
	}

	// Validate port is numeric and in valid range
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port number '%s': %v", port, err)
	}
	if portNum < 1 || portNum > 65535 {
		return fmt.Errorf("port number out of range (1-65535): %d", portNum)
	}

	return nil
}

// UpdateClientConfig updates configurable settings in an existing config file with validation
func UpdateClientConfig(dataDir, configFileName string, serverAddr, grpcCertPath, address, debugLevel, tableTheme, cardTheme string, soundsEnabled, hideTableLogo bool) error {
	configPath := filepath.Join(dataDir, configFileName)

	// Validate inputs
	if err := ValidateCertPath(grpcCertPath); err != nil {
		return fmt.Errorf("invalid cert path: %v", err)
	}

	if err := ValidateAddress(address); err != nil {
		return fmt.Errorf("invalid address/pubkey: %v", err)
	}

	if err := ValidateServerAddress(serverAddr); err != nil {
		return fmt.Errorf("invalid server address: %v", err)
	}

	if err := ValidateTheme(tableTheme, ValidTableThemes, "table"); err != nil {
		return err
	}

	if err := ValidateTheme(cardTheme, ValidCardThemes, "card"); err != nil {
		return err
	}

	// Load existing config
	cfg, err := LoadClientConf(dataDir, configFileName)
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// Update server address (split into host:port)
	if serverAddr != "" {
		host, port, _ := strings.Cut(serverAddr, ":")
		cfg.GRPCHost = host
		cfg.GRPCPort = port
	}

	// Update other fields if provided
	if grpcCertPath != "" {
		cfg.GRPCCertPath = grpcCertPath
	}
	if address != "" {
		cfg.PayoutAddress = address
	}
	if debugLevel != "" {
		cfg.Debug = debugLevel
	}
	if tableTheme != "" {
		cfg.TableTheme = strings.ToLower(strings.TrimSpace(tableTheme))
	}
	if cardTheme != "" {
		cfg.CardTheme = strings.ToLower(strings.TrimSpace(cardTheme))
	}
	cfg.SoundsEnabled = soundsEnabled
	cfg.HideTableLogo = hideTableLogo

	// Write updated config back
	if err := WriteClientConfigFile(cfg, configPath); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	return nil
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
