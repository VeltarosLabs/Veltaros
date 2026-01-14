import { useCallback, useMemo, useState } from "react";
import {
    addressFromPublicKeySpki,
    decryptJsonWithPassword,
    encryptJsonWithPassword,
    exportPrivateKeyPkcs8,
    exportPublicKeySpki,
    extractEd25519RawPublicKeyFromSpki,
    hex,
    importPrivateKeyPkcs8,
    sha256Bytes
} from "../crypto/webcrypto";
import {
    createVaultPayload,
    deleteVault,
    hasEncryptedVault,
    loadEncryptedVault,
    parseVaultPayload,
    saveEncryptedVault,
    type WalletVault
} from "./vault";

export type WalletState =
    | { status: "locked"; hasVault: boolean }
    | {
        status: "unlocked";
        hasVault: true;
        address: string;
        publicKeyFingerprintHex: string;
        publicKeyRawHex: string;
    };

export function useWallet() {
    const [state, setState] = useState<WalletState>(() => ({
        status: "locked",
        hasVault: hasEncryptedVault()
    }));

    const isUnlocked = state.status === "unlocked";

    const refreshHasVault = useCallback(() => {
        setState((s) => {
            if (s.status === "unlocked") return s;
            return { status: "locked", hasVault: hasEncryptedVault() };
        });
    }, []);

    const createNew = useCallback(async (password: string) => {
        if (!password || password.length < 10) throw new Error("Password must be at least 10 characters");

        const kp = await crypto.subtle.generateKey({ name: "Ed25519" }, true, ["sign", "verify"]);
        const publicSpki = await exportPublicKeySpki(kp.publicKey);
        const privatePkcs8 = await exportPrivateKeyPkcs8(kp.privateKey);

        const payload = createVaultPayload(publicSpki, privatePkcs8);
        const encrypted = await encryptJsonWithPassword(payload, password);

        saveEncryptedVault(encrypted);

        const pubFp = hex(await sha256Bytes(publicSpki));
        const addr = await addressFromPublicKeySpki(publicSpki);
        const rawPubHex = hex(extractEd25519RawPublicKeyFromSpki(publicSpki));

        setState({
            status: "unlocked",
            hasVault: true,
            address: addr,
            publicKeyFingerprintHex: pubFp,
            publicKeyRawHex: rawPubHex
        });
    }, []);

    const unlock = useCallback(async (password: string) => {
        const blob = loadEncryptedVault();
        if (!blob) throw new Error("No wallet vault found");

        const payload = await decryptJsonWithPassword<WalletVault>(blob, password);
        const { publicKeySpki } = parseVaultPayload(payload);

        const pubFp = hex(await sha256Bytes(publicKeySpki));
        const addr = await addressFromPublicKeySpki(publicKeySpki);
        const rawPubHex = hex(extractEd25519RawPublicKeyFromSpki(publicKeySpki));

        setState({
            status: "unlocked",
            hasVault: true,
            address: addr,
            publicKeyFingerprintHex: pubFp,
            publicKeyRawHex: rawPubHex
        });
    }, []);

    const lock = useCallback(() => {
        setState({ status: "locked", hasVault: hasEncryptedVault() });
    }, []);

    const reset = useCallback(() => {
        deleteVault();
        setState({ status: "locked", hasVault: false });
    }, []);

    const exportKeysForSigning = useCallback(async (password: string) => {
        const blob = loadEncryptedVault();
        if (!blob) throw new Error("No wallet vault found");

        const payload = await decryptJsonWithPassword<WalletVault>(blob, password);
        const { publicKeySpki, privateKeyPkcs8 } = parseVaultPayload(payload);

        const privKey = await importPrivateKeyPkcs8(privateKeyPkcs8);
        const pubRaw = extractEd25519RawPublicKeyFromSpki(publicKeySpki);

        return { publicKeySpki, privateKeyPkcs8, privateKey: privKey, publicKeyRaw: pubRaw };
    }, []);

    const actions = useMemo(
        () => ({ refreshHasVault, createNew, unlock, lock, reset, exportKeysForSigning }),
        [refreshHasVault, createNew, unlock, lock, reset, exportKeysForSigning]
    );

    return { state, isUnlocked, actions };
}
