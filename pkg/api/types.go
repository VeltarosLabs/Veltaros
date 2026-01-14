package api

type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

type Health struct {
	OK   bool   `json:"ok"`
	Time string `json:"time"`
}

type NodeStatus struct {
	NetworkID   string `json:"networkID"`
	StartedAt   string `json:"startedAt"`
	UptimeSec   int64  `json:"uptimeSec"`
	Peers       int    `json:"peers"`
	KnownPeers  int    `json:"knownPeers"`
	BannedPeers int    `json:"bannedPeers"`
	Height      uint64 `json:"height"`
	DataDir     string `json:"dataDir"`
}

type PeerInfo struct {
	RemoteAddr   string `json:"remoteAddr"`
	Inbound      bool   `json:"inbound"`
	ConnectedAt  int64  `json:"connectedAt"`
	PublicKeyHex string `json:"publicKeyHex"`
	NodeVersion  string `json:"nodeVersion"`
	Verified     bool   `json:"verified"`
	Score        int    `json:"score"`
}

type PeerList struct {
	Count int        `json:"count"`
	Peers []PeerInfo `json:"peers"`
}
