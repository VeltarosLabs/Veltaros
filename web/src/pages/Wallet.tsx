import React, { useMemo, useState } from "react";
import { env } from "../config/env";
import { VeltarosApiClient } from "../api/client";
import type { NodeStatus, PeerList } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useWallet } from "../wallet/useWallet";
import Card from "../components/Card";
import Tabs from "../components/Tabs";
import type { TxDraft, SignedTx } from "../tx/types";
import { signDraft } from "../tx/sign";
import { validateAddress } from "../tx/address";

type Section = "vault" | "tx" | "network";

function formatUptime(seconds: number): string {
    const s = Math.max(0, Math.floor(seconds));
    const d = Math.floor(s / 86400);
    const h = Math.floor((s % 86400) / 3600);
    const m = Math.floor((s % 3600) / 60);
    const r = s % 60;
    const parts: string[] = [];
    if (d) parts.push(`${d}d`);
    if (h || d) parts.push(`${h}h`);
    if (m || h || d) parts.push(`${m}m`);
    parts.push(`${r}s`);
    return parts.join(" ");
}

function tsToIso(ts: number): string {
    const d = new Date(ts * 1000);
    return isNaN(d.getTime()) ? "-" : d.toISOString();
}

export default function Wallet(): React.ReactElement {
    const api = useMemo(() => new VeltarosApiClient(env.nodeApiBaseUrl), []);
    const status = usePoll<NodeStatus>((signal) => api.status(signal), 2500);
    const peers = usePoll<PeerList>((signal) => api.peers(signal), 3000);

    const { state: wallet, actions } = useWallet();

    const [section, setSection] = useState<Section>("vault");
    const [busy, setBusy] = useState(false);
    const [msg, setMsg] = useState<string | null>(null);

    // Vault
    const [pwd, setPwd] = useState("");
    const [pwd2, setPwd2] = useState("");

    // Tx
    const [txTo, setTxTo] = useState("");
    const [txAmount, setTxAmount] = useState("1000");
    const [txFee, setTxFee] = useState("10");
    const [txNonce, setTxNonce] = useState("1");
    const [txMemo, setTxMemo] = useState("");
    const [signed, setSigned] = useState<SignedTx | null>(null);
    const [txResult, setTxResult] = useState<string | null>(null);

    const showError = (e: unknown) => setMsg(e instanceof Error ? e.message : "Unknown error");

    const onCreate = async () => {
        setMsg(null);
        if (pwd !== pwd2) {
            setMsg("Passwords do not match");
            return;
        }
        setBusy(true);
        try {
            await actions.createNew(pwd);
            setPwd("");
            setPwd2("");
        } catch (e) {
            showError(e);
        } finally {
            setBusy(false);
        }
    };

    const onUnlock = async () => {
        setMsg(null);
        setBusy(true);
        try {
            await actions.unlock(pwd);
            setPwd("");
        } catch (e) {
            showError(e);
        } finally {
            setBusy(false);
        }
    };

    const onLock = () => {
        setMsg(null);
        actions.lock();
        setSigned(null);
        setTxResult(null);
    };

    const onReset = () => {
        setMsg(null);
        if (confirm("This will permanently delete the local wallet vault from this browser. Continue?")) {
            actions.reset();
            setSigned(null);
            setTxResult(null);
        }
    };

    const onSignTx = async () => {
        setMsg(null);
        setTxResult(null);
        setBusy(true);
        try {
            if (wallet.status !== "unlocked") throw new Error("Unlock wallet first");
            if (!status.data) throw new Error("Node status not available yet");

            const to = txTo.trim();
            if (!(await validateAddress(to))) {
                throw new Error("Recipient address is invalid (checksum failed or wrong length)");
            }

            const password = prompt("Enter your wallet password to sign this transaction:");
            if (!password) throw new Error("Signing cancelled");

            const { privateKey, publicKeyRaw } = await actions.exportKeysForSigning(password);

            const amount = Number(txAmount);
            const fee = Number(txFee);
            const nonce = Number(txNonce);

            if (!Number.isFinite(amount) || amount <= 0) throw new Error("Invalid amount");
            if (!Number.isFinite(fee) || fee < 1) throw new Error("Fee must be at least 1");
            if (!Number.isFinite(nonce) || nonce <= 0) throw new Error("Invalid nonce");
            if (fee > amount) throw new Error("Fee must be <= amount");
            if (txMemo.length > 256) throw new Error("Memo too long (max 256)");

            const draft: TxDraft = {
                version: 1,
                networkId: status.data.networkID,
                from: wallet.address,
                to,
                amount,
                fee,
                nonce,
                timestamp: Math.floor(Date.now() / 1000),
                memo: txMemo.trim() ? txMemo.trim() : undefined
            };

            const stx = await signDraft(draft, publicKeyRaw, privateKey);
            setSigned(stx);

            const res = await api.txValidate(stx);
            setTxResult(JSON.stringify(res, null, 2));
        } catch (e) {
            showError(e);
        } finally {
            setBusy(false);
        }
    };

    const onBroadcast = async () => {
        setMsg(null);
        setBusy(true);
        try {
            if (!signed) throw new Error("No signed transaction yet");
            const res = await api.txBroadcast(signed);
            setTxResult(JSON.stringify(res, null, 2));
        } catch (e) {
            showError(e);
        } finally {
            setBusy(false);
        }
    };

    const tabs = [
        { key: "vault" as const, label: "Vault" },
        { key: "tx" as const, label: "Transactions" },
        { key: "network" as const, label: "Network" }
    ];

    return (
        <section className="page">
            <div className="pageTop">
                <div>
                    <h2 className="pageTitle">Wallet</h2>
                    <p className="muted">
                        Node API: <code className="code">{env.nodeApiBaseUrl}</code>
                    </p>
                </div>
            </div>

            <Tabs items={tabs} value={section} onChange={setSection} />

            {msg && <div className="alert error">Error: {msg}</div>}

            {section === "vault" && (
                <div className="gridTwo">
                    <Card
                        title="Local Wallet Vault"
                        subtitle="Encrypted key storage (PBKDF2 + AES-GCM). Stored only in this browser."
                        actions={
                            wallet.status === "unlocked" ? (
                                <button className="btn small" onClick={onLock}>
                                    Lock
                                </button>
                            ) : null
                        }
                    >
                        {wallet.status === "locked" && !wallet.hasVault && (
                            <>
                                <p className="muted">No wallet exists locally yet. Create one with a strong password.</p>

                                <label className="label">
                                    Password
                                    <input
                                        className="input"
                                        type="password"
                                        value={pwd}
                                        onChange={(e) => setPwd(e.target.value)}
                                        placeholder="At least 10 characters"
                                        autoComplete="new-password"
                                    />
                                </label>

                                <label className="label">
                                    Confirm Password
                                    <input
                                        className="input"
                                        type="password"
                                        value={pwd2}
                                        onChange={(e) => setPwd2(e.target.value)}
                                        placeholder="Repeat password"
                                        autoComplete="new-password"
                                    />
                                </label>

                                <div className="rowBtns">
                                    <button className="btn primary" onClick={onCreate} disabled={busy}>
                                        Create Wallet
                                    </button>
                                </div>
                            </>
                        )}

                        {wallet.status === "locked" && wallet.hasVault && (
                            <>
                                <p className="muted">Unlock your vault to view address and sign transactions.</p>

                                <label className="label">
                                    Password
                                    <input
                                        className="input"
                                        type="password"
                                        value={pwd}
                                        onChange={(e) => setPwd(e.target.value)}
                                        placeholder="Your wallet password"
                                        autoComplete="current-password"
                                    />
                                </label>

                                <div className="rowBtns">
                                    <button className="btn primary" onClick={onUnlock} disabled={busy}>
                                        Unlock
                                    </button>
                                    <button className="btn danger" onClick={onReset} disabled={busy}>
                                        Delete Vault
                                    </button>
                                </div>
                            </>
                        )}

                        {wallet.status === "unlocked" && (
                            <>
                                <div className="kv">
                                    <div className="row">
                                        <span>Status</span>
                                        <span className="value">Unlocked</span>
                                    </div>
                                    <div className="row">
                                        <span>Address</span>
                                        <span className="value mono">{wallet.address}</span>
                                    </div>
                                    <div className="row">
                                        <span>Public Fingerprint</span>
                                        <span className="value mono">{wallet.publicKeyFingerprintHex.slice(0, 28)}…</span>
                                    </div>
                                </div>

                                <div className="rowBtns">
                                    <button className="btn danger" onClick={onReset}>
                                        Delete Vault
                                    </button>
                                </div>
                            </>
                        )}
                    </Card>

                    <Card title="Node Status" subtitle="Live operational status from your node.">
                        {status.loading && <p className="muted">Loading…</p>}
                        {status.error && <p className="error">Error: {status.error}</p>}

                        {status.data && (
                            <div className="kv">
                                <div className="row">
                                    <span>Network</span>
                                    <span className="value">{status.data.networkID}</span>
                                </div>
                                <div className="row">
                                    <span>Uptime</span>
                                    <span className="value">{formatUptime(status.data.uptimeSec)}</span>
                                </div>
                                <div className="row">
                                    <span>Height</span>
                                    <span className="value">{status.data.height}</span>
                                </div>
                                <div className="row">
                                    <span>Mempool</span>
                                    <span className="value">{status.data.mempool}</span>
                                </div>
                                <div className="row">
                                    <span>Peers</span>
                                    <span className="value">{status.data.peers}</span>
                                </div>
                                <div className="row">
                                    <span>Known</span>
                                    <span className="value">{status.data.knownPeers}</span>
                                </div>
                            </div>
                        )}
                    </Card>
                </div>
            )}

            {section === "tx" && (
                <div className="gridOne">
                    <Card
                        title="Create & Sign Transaction"
                        subtitle="Draft a transaction, sign with your local key, validate and broadcast to the node."
                        actions={
                            <div className="rowBtns">
                                <button className="btn small primary" onClick={onSignTx} disabled={busy || wallet.status !== "unlocked"}>
                                    Sign & Validate
                                </button>
                                <button className="btn small" onClick={onBroadcast} disabled={busy || !signed}>
                                    Broadcast
                                </button>
                            </div>
                        }
                    >
                        <div className="formGrid">
                            <label className="label span2">
                                To (recipient address)
                                <input
                                    className="input mono"
                                    value={txTo}
                                    onChange={(e) => setTxTo(e.target.value)}
                                    placeholder="recipient address (24 bytes hex)"
                                />
                            </label>

                            <label className="label">
                                Amount
                                <input className="input mono" value={txAmount} onChange={(e) => setTxAmount(e.target.value)} />
                            </label>

                            <label className="label">
                                Fee
                                <input className="input mono" value={txFee} onChange={(e) => setTxFee(e.target.value)} />
                            </label>

                            <label className="label">
                                Nonce
                                <input className="input mono" value={txNonce} onChange={(e) => setTxNonce(e.target.value)} />
                            </label>

                            <label className="label span2">
                                Memo (optional, max 256)
                                <input className="input" value={txMemo} onChange={(e) => setTxMemo(e.target.value)} />
                            </label>
                        </div>

                        {signed && (
                            <div className="note">
                                <div className="tiny muted">Signed TX ID</div>
                                <div className="mono">{signed.txId}</div>
                            </div>
                        )}

                        {txResult && <pre className="pre">{txResult}</pre>}
                    </Card>
                </div>
            )}

            {section === "network" && (
                <div className="gridOne">
                    <Card title="Peers" subtitle="Connected peers and verification status.">
                        {peers.loading && <p className="muted">Loading…</p>}
                        {peers.error && <p className="error">Error: {peers.error}</p>}

                        {peers.data && (
                            <>
                                <p className="muted">Connected peers: {peers.data.count}</p>
                                <div className="tableWrap">
                                    <table className="table">
                                        <thead>
                                            <tr>
                                                <th>Remote</th>
                                                <th>Dir</th>
                                                <th>Verified</th>
                                                <th>Score</th>
                                                <th>Version</th>
                                                <th>Connected</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {peers.data.peers.length === 0 ? (
                                                <tr>
                                                    <td colSpan={6} className="muted">
                                                        No peers connected.
                                                    </td>
                                                </tr>
                                            ) : (
                                                peers.data.peers.map((p) => (
                                                    <tr key={p.remoteAddr}>
                                                        <td>
                                                            <div className="mono">{p.remoteAddr}</div>
                                                            <div className="tiny muted mono">{p.publicKeyHex?.slice(0, 20) || "-"}</div>
                                                        </td>
                                                        <td className="mono">{p.inbound ? "in" : "out"}</td>
                                                        <td className="mono">{p.verified ? "yes" : "no"}</td>
                                                        <td className="mono">{p.score}</td>
                                                        <td className="mono">{p.nodeVersion || "-"}</td>
                                                        <td className="mono">{tsToIso(p.connectedAt)}</td>
                                                    </tr>
                                                ))
                                            )}
                                        </tbody>
                                    </table>
                                </div>
                            </>
                        )}
                    </Card>
                </div>
            )}
        </section>
    );
}
