package server

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ServerConf is the config loaded from .conf
type ServerConf struct {
	// Absolute directory where the config/logs live.
	Datadir string

	// Optional path to the JSON file that defines server-managed default tables.
	DefaultTablesPath string

	// Extracted Server gRPC settings (also persisted in BR.ExtraConfig).
	GRPCHost     string
	GRPCPort     string
	GRPCCertPath string
	GRPCKeyPath  string

	// HTTP server TLS settings
	HTTPCertPath   string
	HTTPKeyPath    string
	HTTPCACertPath string // CA certificate to verify client certificates

	DcrdHost string
	DcrdCert string
	DcrdUser string
	DcrdPass string

	AdaptorSecret string
	Network       string

	LogFile        string
	Debug          string
	MaxLogFiles    int
	MaxBufferLines int
}

// parseClientConfigFile parses the config file at the given path into a ClientConfig struct.
func parseClientConfigFile(configPath string, appName string) (*ServerConf, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &ServerConf{}
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
		case "defaulttablespath":
			cfg.DefaultTablesPath = value
		case "grpchost":
			cfg.GRPCHost = value
		case "grpcport":
			cfg.GRPCPort = value
		case "grpccertpath":
			cfg.GRPCCertPath = value
		case "grpckeypath":
			cfg.GRPCKeyPath = value
		case "httpcertpath":
			cfg.HTTPCertPath = value
		case "httpkeypath":
			cfg.HTTPKeyPath = value
		case "httpcacertpath":
			cfg.HTTPCACertPath = value
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
				cfg.MaxLogFiles = 10
			}
		case "maxbufferlines":
			fmt.Sscanf(value, "%d", &cfg.MaxBufferLines)
			if cfg.MaxBufferLines == 0 {
				cfg.MaxBufferLines = 1000
			}
		case "network":
			cfg.Network = value
		case "dcrdhost":
			cfg.DcrdHost = value
		case "dcrdcert":
			cfg.DcrdCert = value
		case "dcrduser":
			cfg.DcrdUser = value
		case "dcrdpass":
			cfg.DcrdPass = value
		case "adaptorsecret":
			cfg.AdaptorSecret = value
		default:
			// Ignore unknown keys to preserve forward-compatibility with older configs.
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var missing []string
	// Check all required fields after parsing (in case keys were missing entirely)
	if strings.TrimSpace(cfg.Datadir) == "" {
		missing = append(missing, "datadir")
	}
	if strings.TrimSpace(cfg.GRPCHost) == "" {
		missing = append(missing, "grpchost")
	}
	if strings.TrimSpace(cfg.GRPCPort) == "" {
		missing = append(missing, "grpcport")
	}
	if strings.TrimSpace(cfg.GRPCCertPath) == "" {
		missing = append(missing, "grpccertpath")
	}
	if strings.TrimSpace(cfg.GRPCKeyPath) == "" {
		missing = append(missing, "grpckeypath")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required fields in client config: %s", strings.Join(missing, ", "))
	}
	if strings.TrimSpace(cfg.DefaultTablesPath) == "" {
		cfg.DefaultTablesPath = filepath.Join(cfg.Datadir, defaultTablesFilename)
	}

	return cfg, nil
}

// loadServerConf attempts to load the client config (.conf) from the default locations.
func loadServerConf(configPath string, fileName string) (*ServerConf, error) {
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
		return parseClientConfigFile(fullPath, appName)
	}

	// Create default config
	cfg := &ServerConf{
		Datadir:           configPath,
		DefaultTablesPath: filepath.Join(configPath, defaultTablesFilename),
		GRPCHost:          "localhost",
		GRPCPort:          "50050",
		GRPCCertPath:      filepath.Join(configPath, "server.cert"),
		GRPCKeyPath:       filepath.Join(configPath, "server.key"),
		HTTPCertPath:      filepath.Join(configPath, "http.cert"),
		HTTPKeyPath:       filepath.Join(configPath, "http.key"),
		HTTPCACertPath:    filepath.Join(configPath, "http-ca.cert"),
		DcrdHost:          "localhost",
		DcrdCert:          filepath.Join(configPath, "dcrd.cert"),
		DcrdUser:          "rpcuser",
		DcrdPass:          "rpcpass",
		Network:           "testnet",
		LogFile:           filepath.Join(configPath, "logs", appName+".log"),
		Debug:             "info",
		MaxLogFiles:       10,
		MaxBufferLines:    1000,
	}

	// Write default config
	if err := writeClientConfigFile(cfg, fullPath); err != nil {
		return nil, err
	}
	if err := ensureDefaultTablesConfigFile(cfg.DefaultTablesPath); err != nil {
		return nil, err
	}

	return cfg, nil
}

// WriteClientConfigFile writes the configuration to a file.
func WriteClientConfigFile(cfg *ServerConf, configPath string) error {
	configData := fmt.Sprintf(
		`datadir=%s
defaulttablespath=%s
grpchost=%s
grpcport=%s
grpccertpath=%s
grpckeypath=%s
httpcertpath=%s
httpkeypath=%s
httpcacertpath=%s
dcrdhost=%s
dcrdcert=%s
dcrduser=%s
dcrdpass=%s
network=%s
logfile=%s
debug=%s
maxlogfiles=%d
maxbufferlines=%d
`,
		cfg.Datadir,
		cfg.DefaultTablesPath,
		cfg.GRPCHost,
		cfg.GRPCPort,
		cfg.GRPCCertPath,
		cfg.GRPCKeyPath,
		cfg.HTTPCertPath,
		cfg.HTTPKeyPath,
		cfg.HTTPCACertPath,
		cfg.DcrdHost,
		cfg.DcrdCert,
		cfg.DcrdUser,
		cfg.DcrdPass,
		cfg.Network,
		cfg.LogFile,
		cfg.Debug,
		cfg.MaxLogFiles,
		cfg.MaxBufferLines,
	)

	return os.WriteFile(configPath, []byte(configData), 0600)
}

// writeClientConfigFile is a wrapper for backward compatibility.
func writeClientConfigFile(cfg *ServerConf, configPath string) error {
	return WriteClientConfigFile(cfg, configPath)
}
