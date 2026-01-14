import type { SignedTx } from "../tx/types";

export type MempoolResponse = {
    count: number;
    txs: SignedTx[];
};
