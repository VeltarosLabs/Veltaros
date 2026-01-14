import React, { useMemo, useState } from "react";
import { env } from "../config/env";
import { VeltarosApiClient } from "../api/client";
import type { NodeStatus, PeerList } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useWallet } from "../wallet/useWallet";

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
    const [pwd, setPwd] = useState("");
    const [pwd2, setPwd2] = useState("");
    const [busy, setBusy] = useState(false);
    const [msg, setMsg] = useState<string | null>(null);

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
    };

    const onReset = () => {
        setMsg(null);
        if (confirm("This will permanently delete the local wallet vault from this browser. Continue?")) {
            actions.reset();
        }
    };

    return (
        <section className="page">
            <h2>Wallet UI</h2>
            <p className="muted">
                Connected to node API: <code className="code">{env.nodeApiBaseUrl}</code>
            </p>

            <div className="grid">
                <div className="card">
                    <h3>Local Wallet Vault</h3>

                    {msg && <p className="error">Error: {msg}</p>}

                    {wallet.status === "locked" && !wallet.hasVault && (
                        <>
                            <p className="muted">
                                No local wallet found on this browser yet. Create one and protect it with a strong password.
                            </p>

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

                            <p className="tiny muted">
                                The private key is encrypted locally using PBKDF2 + AES-GCM and stored in this browser only.
                            </p>
                        </>
                    )}

                    {wallet.status === "locked" && wallet.hasVault && (
                        <>
                            <p className="muted">
                                A local wallet vault exists. Unlock it with your password to view the address and use signing features.
                            </p>

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

                            <p className="tiny muted">
                                If you forgot your password, the vault cannot be recovered. Deleting it will remove the wallet from this
                                browser.
                            </p>
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
                                    <span className="value mono">{wallet.publicKeyHex.slice(0, 24)}…</span>
                                </div>
                            </div>

                            <div className="rowBtns">
                                <button className="btn" onClick={onLock}>
                                    Lock
                                </button>
                                <button className="btn danger" onClick={onReset}>
                                    Delete Vault
                                </button>
                            </div>

                            <p className="tiny muted">
                                Next we’ll add transaction drafting and signing using this keypair, and secure export options.
                            </p>
                        </>
                    )}
                </div>

                <div className="card">
                    <h3>Node Status</h3>

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
                                <span>Peers</span>
                                <span className="value">{status.data.peers}</span>
                            </div>
                            <div className="row">
                                <span>Known Peers</span>
                                <span className="value">{status.data.knownPeers}</span>
                            </div>
                            <div className="row">
                                <span>Banned</span>
                                <span className="value">{status.data.bannedPeers}</span>
                            </div>
                        </div>
                    )}

                    {status.lastUpdated && <p className="tiny muted">Updated: {new Date(status.lastUpdated).toISOString()}</p>}
                </div>
            </div>

            <div className="card">
                <h3>Peers</h3>

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
            </div>
        </section>
    );
}
