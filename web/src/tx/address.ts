import { hexToBytes, doubleSha256Bytes, hex } from "../crypto/webcrypto";

export async function validateAddress(addr: string): Promise<boolean> {
    try {
        const b = hexToBytes(addr.trim());
        if (b.length !== 24) return false;
        const pubHash20 = b.slice(0, 20);
        const got = b.slice(20, 24);
        const want = await doubleSha256Bytes(pubHash20);
        return hex(got) === hex(want.slice(0, 4));
    } catch {
        return false;
    }
}
