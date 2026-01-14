export type VersionInfo = {
    version: string;
    commit: string;
    goVersion: string;
    platform: string;
};

export type Health = {
    ok: boolean;
    time: string;
};

export type NodeStatus = {
    networkID: string;
    startedAt: string;
    uptimeSec: number;
    peers: number;
    knownPeers: number;
    bannedPeers: number;
    height: number;
    dataDir: string;
};

export type PeerInfo = {
    remoteAddr: string;
    inbound: boolean;
    connectedAt: number;
    publicKeyHex: string;
    nodeVersion: string;
    verified: boolean;
    score: number;
};

export type PeerList = {
    count: number;
    peers: PeerInfo[];
};
