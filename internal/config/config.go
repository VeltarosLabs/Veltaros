package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Network NetworkConfig
	API     APIConfig
	Log     LogConfig
	Storage StorageConfig
}

type NetworkConfig struct {
	ListenAddr       string
	ExternalAddr     string
	BootstrapPeers   []string
	MaxPeers         int
	DialTimeout      time.Duration
	HandshakeTimeout time.Duration

	NetworkID          string
	IdentityKeyPath    string
	IdentityRecordPath string
	BanlistPath        string
	PeerStorePath      string
	ScoreStorePath     string
}

type APIConfig struct {
	Enabled      bool
	ListenAddr   string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type LogConfig struct {
	Level  string // debug|info|warn|error
	Format string // json|text
}

type StorageConfig struct {
	DataDir string
}

func Default() Config {
	return Config{
		Network: NetworkConfig{
			ListenAddr:       "0.0.0.0:30303",
			ExternalAddr:     "",
			BootstrapPeers:   []string{},
			MaxPeers:         64,
			DialTimeout:      7 * time.Second,
			HandshakeTimeout: 7 * time.Second,

			NetworkID:          "veltaros-mainnet",
			IdentityKeyPath:    "data/node/identity.key",
			IdentityRecordPath: "data/node/identity.json",
			BanlistPath:        "data/node/banlist.json",
			PeerStorePath:      "data/node/peers.json",
			ScoreStorePath:     "data/node/scores.json",
		},
		API: APIConfig{
			Enabled:      true,
			ListenAddr:   "127.0.0.1:8080",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
		Storage: StorageConfig{
			DataDir: "data",
		},
	}
}

type Parsed struct {
	Config Config
}

func ParseNodeFlags(args []string) (Parsed, error) {
	cfg := Default()

	fs := flag.NewFlagSet("veltaros-node", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	var (
		listenAddr   = fs.String("p2p.listen", envOr("VELTAROS_P2P_LISTEN", cfg.Network.ListenAddr), "P2P listen address (ip:port)")
		externalAddr = fs.String("p2p.external", envOr("VELTAROS_P2P_EXTERNAL", cfg.Network.ExternalAddr), "P2P external address (ip:port) advertised to peers (optional)")
		bootstrap    = fs.String("p2p.bootstrap", envOr("VELTAROS_P2P_BOOTSTRAP", ""), "Comma-separated bootstrap peers (host:port,host:port,...)")
		maxPeers     = fs.Int("p2p.maxPeers", envOrInt("VELTAROS_P2P_MAXPEERS", cfg.Network.MaxPeers), "Maximum connected peers")

		networkID      = fs.String("p2p.network", envOr("VELTAROS_NETWORK_ID", cfg.Network.NetworkID), "Network ID (e.g. veltaros-mainnet, veltaros-testnet)")
		identityKey    = fs.String("p2p.identityKey", envOr("VELTAROS_IDENTITY_KEY", cfg.Network.IdentityKeyPath), "Path to node identity private key (ed25519, hex)")
		identityRecord = fs.String("p2p.identityRecord", envOr("VELTAROS_IDENTITY_RECORD", cfg.Network.IdentityRecordPath), "Path to node identity record JSON (public metadata)")
		banlistPath    = fs.String("p2p.banlist", envOr("VELTAROS_BANLIST_PATH", cfg.Network.BanlistPath), "Path to banlist JSON file")
		peerStore      = fs.String("p2p.peerStore", envOr("VELTAROS_PEERSTORE_PATH", cfg.Network.PeerStorePath), "Path to known peers JSON file")
		scoreStore     = fs.String("p2p.scoreStore", envOr("VELTAROS_SCORESTORE_PATH", cfg.Network.ScoreStorePath), "Path to peer score store JSON file")

		apiEnabled = fs.Bool("api.enabled", envOrBool("VELTAROS_API_ENABLED", cfg.API.Enabled), "Enable HTTP API")
		apiListen  = fs.String("api.listen", envOr("VELTAROS_API_LISTEN", cfg.API.ListenAddr), "HTTP API listen address (ip:port)")

		logLevel  = fs.String("log.level", envOr("VELTAROS_LOG_LEVEL", cfg.Log.Level), "Log level: debug|info|warn|error")
		logFormat = fs.String("log.format", envOr("VELTAROS_LOG_FORMAT", cfg.Log.Format), "Log format: json|text")

		dataDir = fs.String("data.dir", envOr("VELTAROS_DATA_DIR", cfg.Storage.DataDir), "Data directory for node storage")
	)

	if err := fs.Parse(args); err != nil {
		return Parsed{}, err
	}

	cfg.Network.ListenAddr = strings.TrimSpace(*listenAddr)
	cfg.Network.ExternalAddr = strings.TrimSpace(*externalAddr)
	cfg.Network.MaxPeers = *maxPeers

	cfg.Network.NetworkID = strings.TrimSpace(*networkID)
	cfg.Network.IdentityKeyPath = strings.TrimSpace(*identityKey)
	cfg.Network.IdentityRecordPath = strings.TrimSpace(*identityRecord)
	cfg.Network.BanlistPath = strings.TrimSpace(*banlistPath)
	cfg.Network.PeerStorePath = strings.TrimSpace(*peerStore)
	cfg.Network.ScoreStorePath = strings.TrimSpace(*scoreStore)

	cfg.API.Enabled = *apiEnabled
	cfg.API.ListenAddr = strings.TrimSpace(*apiListen)
	cfg.Log.Level = strings.TrimSpace(*logLevel)
	cfg.Log.Format = strings.TrimSpace(*logFormat)
	cfg.Storage.DataDir = strings.TrimSpace(*dataDir)

	if b := strings.TrimSpace(*bootstrap); b != "" {
		cfg.Network.BootstrapPeers = splitCSV(b)
	}

	if err := validate(cfg); err != nil {
		return Parsed{}, err
	}

	return Parsed{Config: cfg}, nil
}

func validate(cfg Config) error {
	if cfg.Network.ListenAddr == "" {
		return errors.New("p2p.listen must not be empty")
	}
	if cfg.Network.MaxPeers <= 0 || cfg.Network.MaxPeers > 4096 {
		return fmt.Errorf("p2p.maxPeers out of range: %d", cfg.Network.MaxPeers)
	}
	if cfg.Network.NetworkID == "" {
		return errors.New("p2p.network must not be empty")
	}
	if cfg.Network.IdentityKeyPath == "" {
		return errors.New("p2p.identityKey must not be empty")
	}
	if cfg.Network.IdentityRecordPath == "" {
		return errors.New("p2p.identityRecord must not be empty")
	}
	if cfg.Network.BanlistPath == "" {
		return errors.New("p2p.banlist must not be empty")
	}
	if cfg.Network.PeerStorePath == "" {
		return errors.New("p2p.peerStore must not be empty")
	}
	if cfg.Network.ScoreStorePath == "" {
		return errors.New("p2p.scoreStore must not be empty")
	}

	switch strings.ToLower(cfg.Log.Level) {
	case "debug", "info", "warn", "warning", "error":
	default:
		return fmt.Errorf("invalid log.level: %q", cfg.Log.Level)
	}

	switch strings.ToLower(cfg.Log.Format) {
	case "json", "text":
	default:
		return fmt.Errorf("invalid log.format: %q", cfg.Log.Format)
	}

	if cfg.API.Enabled && cfg.API.ListenAddr == "" {
		return errors.New("api.listen must not be empty when api.enabled=true")
	}
	if cfg.Storage.DataDir == "" {
		return errors.New("data.dir must not be empty")
	}
	return nil
}

func envOr(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func envOrInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envOrBool(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}

func splitCSV(s string) []string {
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		t := strings.TrimSpace(r)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}
