import React, { useMemo, useState } from "react";
import { env } from "../config/env";
import { VeltarosApiClient, type TxBroadcastResponse, type TxValidateResponse } from "../api/client";
import type { NodeStatus, PeerList } from "../api/types";
import type { MempoolResponse } from "../api/mempoolTypes";
import { usePoll } from "../hooks/usePoll";
import { useWallet } from "../wallet/useWallet";
import Card from "../components/Card";
import Tabs from "../components/Tabs";
import Modal from "../components/Modal";
import type { TxDraft, SignedTx } from "../tx/types";
import { signDraft } from "../tx/sign";
import { validateAddress } from "../tx/address";
import { clearHistory, loadHistory, upsertHistory, type TxHistoryItem } from "../tx/history";
import "../styles/wallet.css";

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

async function copyToClipboard(text: string): Promise<void> {
    await navigator.clipboard.writeText(text);
}

function asNumberString(n: number | undefined): string | null {
    if (typeof n !== "number") return null;
    if (!Number.isFinite(n)) return null;
    return String(Math.max(0, Math.floor(n)));
}

export default function Wallet(): React.ReactElement {
    const api = useMemo(() => new VeltarosApiClient(env.nodeApiBaseUrl), []);
    const status = usePoll<NodeStatus>((signal) => api.status(signal), 2500);
    const peers = usePoll<PeerList>((signal) => api.peers(signal), 3000);
    const mempool = usePoll<MempoolResponse>((signal) => api.mempool(signal), 3500);

    const { state: wallet, actions } = useWallet();

    const [section, setSection] = useState<Section>("vault");
    const [busy, setBusy] = useState(false);
    const [notice, setNotice] = useState<string | null>(null);

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

    // Signing modal
    const [signOpen, setSignOpen] = useState(false);
    const [signPwd, setSignPwd] = useState("");

    // History
    const [history, setHistory] = useState<TxHistoryItem[]>(() => loadHistory());
    const refreshHistory = () => setHistory(loadHistory());

    const tabs = [
        { key: "vault" as const, label: "Vault" },
        { key: "tx" as const, label: "Transactions", badge: history.length ? String(history.length) : undefined },
        { key: "network" as const, label: "Network" }
    ];

    const show = (text: string) => {
        setNotice(text);
        window.setTimeout(() => setNotice(null), 1800);
    };

    const fail = (e: unknown) => {
        const msg = e instanceof Error ? e.message : "Something went wrong";
        setNotice(msg);
    };

    const onCopy = async (text: string) => {
        try {
            await copyToClipboard(text);
            show("Copied");
        } catch {
            setNotice("Clipboard copy failed");
        }
    };

    const onCreate = async () => {
        setNotice(null);
        if (pwd !== pwd2) {
            setNotice("Passwords do not match");
            return;
        }
        setBusy(true);
        try {
            await actions.createNew(pwd);
            setPwd("");
            setPwd2("");
            show("Wallet created");
        } catch (e) {
            fail(e);
        } finally {
            setBusy(false);
        }
    };

    const onUnlock = async () => {
        setNotice(null);
        setBusy(true);
        try {
            await actions.unlock(pwd);
            setPwd("");
            show("Unlocked");
        } catch (e) {
            fail(e);
        } finally {
            setBusy(false);
        }
    };

    const onLock = () => {
        actions.lock();
        setSigned(null);
        setTxResult(null);
        show("Locked");
    };

    const onReset = () => {
        if (confirm("Delete the local wallet vault from this device? This cannot be undone.")) {
            actions.reset();
            setSigned(null);
            setTxResult(null);
            show("Vault deleted");
        }
    };

    const openSign = async () => {
        setNotice(null);
        setTxResult(null);

        if (wallet.status !== "unlocked") {
            setNotice("Unlock your wallet first");
            return;
        }
        if (!status.data) {
            setNotice("Node status is not ready yet");
            return;
        }

        const to = txTo.trim();
        if (!(await validateAddress(to))) {
            setNotice("Recipient address is invalid");
            return;
        }

        const amount = Number(txAmount);
        const fee = Number(txFee);
        const nonce = Number(txNonce);

        if (!Number.isFinite(amount) || amount <= 0) {
            setNotice("Amount must be greater than 0");
            return;
        }
        if (!Number.isFinite(fee) || fee < 1) {
            setNotice("Fee must be at least 1");
            return;
        }
        if (fee > amount) {
            setNotice("Fee must be less than or equal to amount");
            return;
        }
        if (!Number.isFinite(nonce) || nonce <= 0) {
            setNotice("Nonce must be greater than 0");
            return;
        }
        if (txMemo.length > 256) {
            setNotice("Memo is too long (max 256)");
            return;
        }

        // Optional: ask node for expected nonce based on current state
        try {
            const v: TxValidateResponse = await api.txValidate({
                draft: {
                    version: 1,
                    networkId: status.data.networkID,
                    from: wallet.address,
                    to,
                    amount,
                    fee,
                    nonce,
                    timestamp: Math.floor(Date.now() / 1000),
                    memo: txMemo.trim() ? txMemo.trim() : undefined
                },
                publicKeyHex: wallet.publicKeyRawHex,
                signatureHex: "00".repeat(64), // placeholder, server will fail signature but will still return nonce data only if ok; so ignore here
                txId: "00".repeat(32)
            } as unknown as SignedTx);

            if ("ok" in v && v.ok && typeof v.expectedNonce === "number") {
                const suggested = asNumberString(v.expectedNonce);
                if (suggested) setTxNonce(suggested);
            }
        } catch {
            // ignore; we only use this as a convenience
        }

        setSignPwd("");
        setSignOpen(true);
    };

    const onSignConfirm = async () => {
        setBusy(true);
        setNotice(null);
        setTxResult(null);

        try {
            if (wallet.status !== "unlocked") throw new Error("Unlock your wallet first");
            if (!status.data) throw new Error("Node status is not ready yet");

            const to = txTo.trim();
            const { privateKey, publicKeyRaw } = await actions.exportKeysForSigning(signPwd);

            const draft: TxDraft = {
                version: 1,
                networkId: status.data.networkID,
                from: wallet.address,
                to,
                amount: Number(txAmount),
                fee: Number(txFee),
                nonce: Number(txNonce),
                timestamp: Math.floor(Date.now() / 1000),
                memo: txMemo.trim() ? txMemo.trim() : undefined
            };

            const stx = await signDraft(draft, publicKeyRaw, privateKey);
            setSigned(stx);

            const base: TxHistoryItem = {
                id: stx.txId,
                createdAt: new Date().toISOString(),
                status: "drafted",
                tx: stx
            };

            upsertHistory(base);
            refreshHistory();

            const res: TxValidateResponse = await api.txValidate(stx);
            setTxResult(JSON.stringify(res, null, 2));

            if ("ok" in res && res.ok) {
                upsertHistory({ ...base, status: "validated", note: `Validated. Expected nonce: ${res.expectedNonce}` });
                refreshHistory();

                const suggested = asNumberString(res.expectedNonce);
                if (suggested) setTxNonce(suggested);

                setSignOpen(false);
                setSignPwd("");
                show("Signed and validated");
                return;
            }

            upsertHistory({ ...base, status: "error", note: res.error });
            refreshHistory();
            setNotice(res.error);
        } catch (e) {
            fail(e);
        } finally {
            setBusy(false);
        }
    };

    const onBroadcast = async () => {
        setBusy(true);
        setNotice(null);

        try {
            if (!signed) throw new Error("No signed transaction yet");

            const res: TxBroadcastResponse = await api.txBroadcast(signed);
            setTxResult(JSON.stringify(res, null, 2));

            if ("ok" in res && res.ok) {
                upsertHistory({
                    id: signed.txId,
                    createdAt: new Date().toISOString(),
                    status: "broadcast",
                    note: res.note ? res.note : "Accepted by node mempool",
                    tx: signed
                });
                refreshHistory();
                show(res.note ? res.note : "Broadcast accepted");
                return;
            }

            // If nonce guidance is present, help the user
            if (typeof res.expectedNonce === "number") {
                const suggested = asNumberString(res.expectedNonce);
                if (suggested) {
                    setTxNonce(suggested);
                    setNotice(`${res.error}. Suggested nonce: ${suggested}`);
                } else {
                    setNotice(res.error);
                }
            } else {
                setNotice(res.error);
            }
        } catch (e) {
            fail(e);
        } finally {
            setBusy(false);
        }
    };

    const onClearHistory = () => {
        if (confirm("Clear local transaction history on this device?")) {
            clearHistory();
            refreshHistory();
            show("History cleared");
        }
    };

    return (
        <section className="page">
            <div className="pageTop">
                <div>
                    <h2 className="pageTitle">Wallet</h2>
                    <p className="muted">
                        Connected node: <code className="code">{env.nodeApiBaseUrl}</code>
                    </p>
                </div>
            </div>

            <Tabs items={tabs} value={section} onChange={setSection} />

            {notice && <div className="alert">{notice}</div>}

            {section === "vault" && (
                <div className="walletGrid">
                    <Card
                        title="Vault"
                        subtitle="Your keys are encrypted and stored locally on this device."
                        actions={wallet.status === "unlocked" ? <button className="btn small" onClick={onLock}>Lock</button> : null}
                    >
                        {wallet.status === "locked" && !wallet.hasVault && (
                            <>
                                <p className="walletHint">Create a local wallet to receive and send Veltaros.</p>

                                <label className="label">
                                    Password
                                    <input className="input" type="password" value={pwd} onChange={(e) => setPwd(e.target.value)} autoComplete="new-password" />
                                </label>

                                <label className="label">
                                    Confirm password
                                    <input className="input" type="password" value={pwd2} onChange={(e) => setPwd2(e.target.value)} autoComplete="new-password" />
                                </label>

                                <div className="rowBtns">
                                    <button className="btn primary" onClick={onCreate} disabled={busy}>Create wallet</button>
                                </div>
                            </>
                        )}

                        {wallet.status === "locked" && wallet.hasVault && (
                            <>
                                <p className="walletHint">Unlock to view your address and sign transactions.</p>

                                <label className="label">
                                    Password
                                    <input className="input" type="password" value={pwd} onChange={(e) => setPwd(e.target.value)} autoComplete="current-password" />
                                </label>

                                <div className="rowBtns">
                                    <button className="btn primary" onClick={onUnlock} disabled={busy}>Unlock</button>
                                    <button className="btn danger" onClick={onReset} disabled={busy}>Delete</button>
                                </div>
                            </>
                        )}

                        {wallet.status === "unlocked" && (
                            <>
                                <div className="kv">
                                    <div className="row">
                                        <span>Address</span>
                                        <span className="value mono">{wallet.address}</span>
                                    </div>
                                    <div className="row">
                                        <span>Fingerprint</span>
                                        <span className="value mono">{wallet.publicKeyFingerprintHex.slice(0, 28)}…</span>
                                    </div>
                                </div>

                                <div className="rowBtns">
                                    <button className="btn" onClick={() => void onCopy(wallet.address)}>Copy address</button>
                                    <button className="btn danger" onClick={onReset}>Delete</button>
                                </div>
                            </>
                        )}
                    </Card>

                    <Card title="Receive" subtitle="Share your address to receive Veltaros.">
                        {wallet.status !== "unlocked" ? (
                            <p className="walletHint">Unlock your wallet to show your receive address.</p>
                        ) : (
                            <>
                                <div className="note">
                                    <div className="tiny muted">Your address</div>
                                    <div className="mono">{wallet.address}</div>
                                </div>

                                <div className="rowBtns">
                                    <button className="btn primary" onClick={() => void onCopy(wallet.address)}>Copy</button>
                                </div>
                            </>
                        )}
                    </Card>
                </div>
            )}

            {section === "tx" && (
                <div className="walletGrid">
                    <Card
                        title="Send"
                        subtitle="Create a transaction draft, then sign with your vault."
                        actions={
                            <div className="rowBtns">
                                <button className="btn small primary" onClick={() => void openSign()} disabled={busy || wallet.status !== "unlocked"}>
                                    Sign
                                </button>
                                <button className="btn small" onClick={() => void onBroadcast()} disabled={busy || !signed}>
                                    Broadcast
                                </button>
                            </div>
                        }
                    >
                        <div className="formGrid2">
                            <label className="label span2">
                                Recipient address
                                <input className="input mono" value={txTo} onChange={(e) => setTxTo(e.target.value)} placeholder="recipient address" />
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
                                Memo (optional)
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

                    <Card
                        title="History"
                        subtitle="Stored locally on this device."
                        actions={<button className="btn small danger" onClick={onClearHistory} disabled={history.length === 0}>Clear</button>}
                    >
                        {history.length === 0 ? (
                            <p className="muted">No transactions yet.</p>
                        ) : (
                            <div className="history">
                                {history.map((h) => (
                                    <button
                                        key={h.id}
                                        type="button"
                                        className="historyItem"
                                        onClick={() => {
                                            setSigned(h.tx);
                                            setTxResult(JSON.stringify(h, null, 2));
                                            show("Loaded transaction");
                                        }}
                                    >
                                        <div className="historyTop">
                                            <span className={`chip ${h.status}`.trim()}>{h.status}</span>
                                            <span className="tiny muted">{h.createdAt}</span>
                                        </div>
                                        <div className="mono">{h.id}</div>
                                        {h.note && <div className="tiny muted">{h.note}</div>}
                                    </button>
                                ))}
                            </div>
                        )}
                    </Card>
                </div>
            )}

            {section === "network" && (
                <div className="walletGrid spanAll">
                    <Card title="Node" subtitle="Live node status from your running Veltaros node.">
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
                            </div>
                        )}
                    </Card>

                    <Card title="Mempool" subtitle="Transactions accepted by the node.">
                        {mempool.loading && <p className="muted">Loading…</p>}
                        {mempool.error && <p className="error">Error: {mempool.error}</p>}

                        {mempool.data && (
                            <>
                                <p className="muted">Count: {mempool.data.count}</p>

                                {mempool.data.txs.length === 0 ? (
                                    <p className="muted">No transactions in mempool.</p>
                                ) : (
                                    <div className="history">
                                        {mempool.data.txs.slice(0, 30).map((t) => (
                                            <button
                                                key={t.txId}
                                                type="button"
                                                className="historyItem"
                                                onClick={() => {
                                                    setSigned(t);
                                                    setTxResult(JSON.stringify(t, null, 2));
                                                    setSection("tx");
                                                    show("Loaded from mempool");
                                                }}
                                            >
                                                <div className="historyTop">
                                                    <span className="chip validated">mempool</span>
                                                    <span className="tiny muted">nonce: {t.draft.nonce}</span>
                                                </div>
                                                <div className="mono">{t.txId}</div>
                                                <div className="tiny muted mono">
                                                    {t.draft.from.slice(0, 12)}… → {t.draft.to.slice(0, 12)}…
                                                </div>
                                            </button>
                                        ))}
                                    </div>
                                )}
                            </>
                        )}
                    </Card>

                    <Card title="Peers" subtitle="Connected peers and connection details.">
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
                                                    <td colSpan={6} className="muted">No peers connected.</td>
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

            <Modal open={signOpen} title="Sign transaction" onClose={() => setSignOpen(false)}>
                <p className="muted">Enter your password to unlock the vault and sign this transaction.</p>

                <label className="label">
                    Wallet password
                    <input className="input" type="password" value={signPwd} onChange={(e) => setSignPwd(e.target.value)} autoComplete="current-password" />
                </label>

                <div className="rowBtns">
                    <button className="btn primary" onClick={() => void onSignConfirm()} disabled={busy || !signPwd}>
                        Sign & Validate
                    </button>
                    <button className="btn" onClick={() => setSignOpen(false)} disabled={busy}>
                        Cancel
                    </button>
                </div>
            </Modal>
        </section>
    );
}
