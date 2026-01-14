import type { EncryptedBlob } from "../crypto/webcrypto";

const STORAGE_KEY = "veltaros.wallet.v1";

export type WalletVault = {
    v: 1;
    createdAt: string;
    updatedAt: string;
    key: {
        // SPKI and PKCS8 formats for portability
        publicKeySpkiHex: string;
        privateKeyPkcs8Hex: string;
    };
};

function hexToBytes(h: string): Uint8Array {
    const s = h.trim().toLowerCase();
    if (s.length % 2 !== 0) throw new Error("Invalid hex length");
    const out = new Uint8Array(s.length / 2);
    for (let i = 0; i < out.length; i++) {
        out[i] = parseInt(s.slice(i * 2, i * 2 + 2), 16);
    }
    return out;
}

function bytesToHex(b: Uint8Array): string {
    return Array.from(b)
        .map((x) => x.toString(16).padStart(2, "0"))
        .join("");
}

export function hasEncryptedVault(): boolean {
    return Boolean(localStorage.getItem(STORAGE_KEY));
}

export function loadEncryptedVault(): EncryptedBlob | null {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as EncryptedBlob;
}

export function saveEncryptedVault(blob: EncryptedBlob): void {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(blob));
}

export function deleteVault(): void {
    localStorage.removeItem(STORAGE_KEY);
}

export function createVaultPayload(publicKeySpki: Uint8Array, privateKeyPkcs8: Uint8Array): WalletVault {
    const now = new Date().toISOString();
    return {
        v: 1,
        createdAt: now,
        updatedAt: now,
        key: {
            publicKeySpkiHex: bytesToHex(publicKeySpki),
            privateKeyPkcs8Hex: bytesToHex(privateKeyPkcs8)
        }
    };
}

export function parseVaultPayload(v: WalletVault): { publicKeySpki: Uint8Array; privateKeyPkcs8: Uint8Array } {
    if (!v || v.v !== 1) throw new Error("Unsupported vault version");
    return {
        publicKeySpki: hexToBytes(v.key.publicKeySpkiHex),
        privateKeyPkcs8: hexToBytes(v.key.privateKeyPkcs8Hex)
    };
}
