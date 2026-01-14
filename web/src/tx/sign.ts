import { hex, sha256Bytes, signEd25519 } from "../crypto/webcrypto";
import type { TxDraft, SignedTx } from "./types";

// Canonical JSON bytes like Go's json.Marshal on struct order:
// We keep stable field order by constructing an object with explicit keys in order.
function canonicalDraftObject(d: TxDraft) {
    return {
        version: d.version,
        networkId: d.networkId,
        from: d.from,
        to: d.to,
        amount: d.amount,
        fee: d.fee,
        nonce: d.nonce,
        timestamp: d.timestamp,
        memo: d.memo ?? ""
    };
}

export async function txHashHex(draft: TxDraft): Promise<string> {
    const bytes = new TextEncoder().encode(JSON.stringify(canonicalDraftObject(draft)));
    const h1 = await sha256Bytes(bytes);
    const h2 = await sha256Bytes(h1);
    return hex(h2);
}

export async function signatureMessage32(networkId: string, txHashHexStr: string): Promise<Uint8Array> {
    const domain = new TextEncoder().encode("veltaros-tx-sign");
    const nid = new TextEncoder().encode(networkId);

    const txHashBytes = new Uint8Array(txHashHexStr.length / 2);
    for (let i = 0; i < txHashBytes.length; i++) {
        txHashBytes[i] = parseInt(txHashHexStr.slice(i * 2, i * 2 + 2), 16);
    }

    const msg = new Uint8Array(domain.length + nid.length + txHashBytes.length);
    msg.set(domain, 0);
    msg.set(nid, domain.length);
    msg.set(txHashBytes, domain.length + nid.length);

    // Go uses SHA256(domain||networkID||txHash)
    return sha256Bytes(msg);
}

export async function signDraft(
    draft: TxDraft,
    publicKeyRaw: Uint8Array,
    privateKey: CryptoKey
): Promise<SignedTx> {
    const txId = await txHashHex(draft);
    const msg32 = await signatureMessage32(draft.networkId, txId);
    const sig = await signEd25519(privateKey, msg32);

    return {
        draft,
        publicKeyHex: hex(publicKeyRaw),
        signatureHex: hex(sig),
        txId
    };
}
