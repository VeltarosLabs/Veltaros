export type Health = {
    ok: boolean;
    time: string;
};

export type VersionInfo = {
    name: string;
    version: string;
    commit: string;
    buildTime?: string;
};

export type NodeStatus = {
    networkID: string;
    startedAt: string;
    uptimeSec: number;

    height: number;
    mempool: number;

    peers: number;
    knownPeers?: number;
    bannedPeers?: number;

    dataDir?: string;
    devMode?: boolean;
};

export type PeerInfo = {
    remoteAddr: string;
    inbound: boolean;
    verified: boolean;
    score: number;
    publicKeyHex?: string;
    nodeVersion?: string;
    connectedAt: number;
};

export type PeerList = {
    count: number;
    peers: PeerInfo[];
};
