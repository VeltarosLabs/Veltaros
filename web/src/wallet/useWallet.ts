import { useCallback, useMemo, useState } from "react";
import {
    addressFromPublicKeyRaw,
    decryptJsonWithPassword,
    encryptJsonWithPassword,
    exportPrivateKeyPkcs8,
    exportPublicKeyRaw,
    generateEd25519Keypair,
    hex,
    importPrivateKeyPkcs8,
    importPublicKeySpki,
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
        publicKeyHex: string;
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
        if (!password || password.length < 10) {
            throw new Error("Password must be at least 10 characters");
        }

        const kp = await generateEd25519Keypair();
        const publicSpki = await exportPublicKeyRaw(kp.publicKey);
        const privatePkcs8 = await exportPrivateKeyPkcs8(kp.privateKey);

        const payload = createVaultPayload(publicSpki, privatePkcs8);
        const encrypted = await encryptJsonWithPassword(payload, password);

        saveEncryptedVault(encrypted);

        // Derive identity for UI
        const pubHash = await sha256Bytes(publicSpki);
        const pubHex = hex(pubHash); // display-friendly stable fingerprint (hash of SPKI)
        const addr = await addressFromPublicKeyRaw(publicSpki);

        setState({ status: "unlocked", hasVault: true, address: addr, publicKeyHex: pubHex });
    }, []);

    const unlock = useCallback(async (password: string) => {
        const blob = loadEncryptedVault();
        if (!blob) throw new Error("No wallet vault found");

        const payload = await decryptJsonWithPassword<WalletVault>(blob, password);
        const { publicKeySpki } = parseVaultPayload(payload);

        const pubHash = await sha256Bytes(publicKeySpki);
        const pubHex = hex(pubHash);
        const addr = await addressFromPublicKeyRaw(publicKeySpki);

        setState({ status: "unlocked", hasVault: true, address: addr, publicKeyHex: pubHex });
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

        const pubKey = await importPublicKeySpki(publicKeySpki);
        const privKey = await importPrivateKeyPkcs8(privateKeyPkcs8);

        return { publicKeySpki, privateKeyPkcs8, pubKey, privKey };
    }, []);

    const actions = useMemo(
        () => ({
            refreshHasVault,
            createNew,
            unlock,
            lock,
            reset,
            exportKeysForSigning
        }),
        [refreshHasVault, createNew, unlock, lock, reset, exportKeysForSigning]
    );

    return { state, isUnlocked, actions };
}
