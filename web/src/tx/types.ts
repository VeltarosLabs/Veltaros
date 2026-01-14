export type TxDraft = {
    version: number; // must be 1
    networkId: string;

    from: string;
    to: string;

    amount: number;
    fee: number;

    nonce: number;
    timestamp: number;

    memo?: string;
};

export type SignedTx = {
    draft: TxDraft;
    publicKeyHex: string; // hex of sha256(spki) OR real pubkey? we'll use pubkey SPKI hash? -> backend expects 32-byte ed25519 pubkey hex
    signatureHex: string;
    txId: string;
};

export type TxValidateResponse = { ok: true; txId: string } | { ok: false; error: string };
export type TxBroadcastResponse = { ok: true; txId: string } | { ok: false; error: string };
