import type { SignedTx } from "../tx/types";

export type TipInfo = {
    height: number;
    tipHash: string;
};

export type NodeStatusLite = {
    devMode?: boolean;
    height?: number;
    mempool?: number;
    tipHash?: string;
    networkID?: string;
};

export type StoredBlockSummary = {
    hash: string;
    height: number;
    prevHash: string;
    merkleRoot: string;
    timestamp: number;
    txCount: number;
};

export type StoredBlock = StoredBlockSummary & {
    block: {
        header: {
            version: number;
            prevHash: string;
            merkleRoot: string;
            timestamp: number;
            nonce: number;
        };
        transactions: SignedTx[];
    };
};

export type BlocksResponse = {
    count: number;
    blocks: StoredBlockSummary[];
};
