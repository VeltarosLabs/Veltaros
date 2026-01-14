export type KdfParams = {
    saltB64: string;
    iterations: number;
    hash: "SHA-256";
};

export type EncryptedBlob = {
    v: 1;
    kdf: KdfParams;
    ivB64: string;
    ctB64: string;
};

const enc = new TextEncoder();
const dec = new TextDecoder();

function toB64(bytes: Uint8Array): string {
    let s = "";
    for (let i = 0; i < bytes.length; i++) s += String.fromCharCode(bytes[i]);
    return btoa(s);
}

function fromB64(b64: string): Uint8Array {
    const bin = atob(b64);
    const out = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
    return out;
}

function concatBytes(a: Uint8Array, b: Uint8Array): Uint8Array {
    const out = new Uint8Array(a.length + b.length);
    out.set(a, 0);
    out.set(b, a.length);
    return out;
}

export async function sha256Bytes(data: Uint8Array): Promise<Uint8Array> {
    const digest = await crypto.subtle.digest("SHA-256", data);
    return new Uint8Array(digest);
}

export async function doubleSha256Bytes(data: Uint8Array): Promise<Uint8Array> {
    const first = await sha256Bytes(data);
    return sha256Bytes(first);
}

export function hex(bytes: Uint8Array): string {
    return Array.from(bytes)
        .map((b) => b.toString(16).padStart(2, "0"))
        .join("");
}

export async function addressFromPublicKeyRaw(publicKeyRaw: Uint8Array): Promise<string> {
    // pubHash20 = sha256(pubKey)[:20]
    const h = await sha256Bytes(publicKeyRaw);
    const pubHash20 = h.slice(0, 20);
    // checksum4 = doubleSha256(pubHash20)[:4]
    const chk = await doubleSha256Bytes(pubHash20);
    const addr = concatBytes(pubHash20, chk.slice(0, 4));
    return hex(addr);
}

export async function generateEd25519Keypair(): Promise<CryptoKeyPair> {
    return crypto.subtle.generateKey(
        { name: "Ed25519" },
        true,
        ["sign", "verify"]
    );
}

export async function exportPublicKeyRaw(key: CryptoKey): Promise<Uint8Array> {
    const spki = await crypto.subtle.exportKey("spki", key);
    // For Ed25519, SPKI wraps the raw pubkey. We keep SPKI for portability.
    return new Uint8Array(spki);
}

export async function exportPrivateKeyPkcs8(key: CryptoKey): Promise<Uint8Array> {
    const pkcs8 = await crypto.subtle.exportKey("pkcs8", key);
    return new Uint8Array(pkcs8);
}

export async function importPrivateKeyPkcs8(pkcs8: Uint8Array): Promise<CryptoKey> {
    return crypto.subtle.importKey(
        "pkcs8",
        pkcs8,
        { name: "Ed25519" },
        true,
        ["sign"]
    );
}

export async function importPublicKeySpki(spki: Uint8Array): Promise<CryptoKey> {
    return crypto.subtle.importKey(
        "spki",
        spki,
        { name: "Ed25519" },
        true,
        ["verify"]
    );
}

export async function deriveAesKeyFromPassword(password: string, salt: Uint8Array, iterations: number): Promise<CryptoKey> {
    const baseKey = await crypto.subtle.importKey(
        "raw",
        enc.encode(password),
        "PBKDF2",
        false,
        ["deriveKey"]
    );

    return crypto.subtle.deriveKey(
        {
            name: "PBKDF2",
            salt,
            iterations,
            hash: "SHA-256"
        },
        baseKey,
        { name: "AES-GCM", length: 256 },
        false,
        ["encrypt", "decrypt"]
    );
}

export async function encryptJsonWithPassword(obj: unknown, password: string): Promise<EncryptedBlob> {
    const salt = crypto.getRandomValues(new Uint8Array(16));
    const iv = crypto.getRandomValues(new Uint8Array(12));
    const iterations = 310_000;

    const key = await deriveAesKeyFromPassword(password, salt, iterations);

    const plaintext = enc.encode(JSON.stringify(obj));
    const ct = await crypto.subtle.encrypt({ name: "AES-GCM", iv }, key, plaintext);

    return {
        v: 1,
        kdf: { saltB64: toB64(salt), iterations, hash: "SHA-256" },
        ivB64: toB64(iv),
        ctB64: toB64(new Uint8Array(ct))
    };
}

export async function decryptJsonWithPassword<T>(blob: EncryptedBlob, password: string): Promise<T> {
    if (!blob || blob.v !== 1) throw new Error("Unsupported vault format");

    const salt = fromB64(blob.kdf.saltB64);
    const iv = fromB64(blob.ivB64);
    const ct = fromB64(blob.ctB64);

    const key = await deriveAesKeyFromPassword(password, salt, blob.kdf.iterations);

    const pt = await crypto.subtle.decrypt({ name: "AES-GCM", iv }, key, ct);
    const text = dec.decode(pt);
    return JSON.parse(text) as T;
}
