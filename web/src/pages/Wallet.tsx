import React, { useMemo } from "react";
import { env } from "../config/env";
import { VeltarosApiClient } from "../api/client";
import type { NodeStatus, PeerList } from "../api/types";
import { usePoll } from "../hooks/usePoll";

function formatUptime(seconds: number): string {
    const s = Math.max(0, Math.floor(seconds));
    const d = Math.floor(s / 86400);
    const h = Math.floor((s % 86400) / 3600);
    const m = Math.floor((s % 3600) / 60);
    const r = s % 60;
    const parts = [];
    if (d) parts.push(`${d}d`);
    if (h || d) parts.push(`${h}h`);
    if (m || h || d) parts.push(`${m}m`);
    parts.push(`${r}s`);
    return parts.join(" ");
}

function tsToIso(ts: number): string {
    // connectedAt is unix seconds
    const d = new Date(ts * 1000);
    return isNaN(d.getTime()) ? "-" : d.toISOString();
}

export default function Wallet(): React.ReactElement {
    const api = useMemo(() => new VeltarosApiClient(env.nodeApiBaseUrl), []);

    const status = usePoll<NodeStatus>((signal) => api.status(signal), 2500);
    const peers = usePoll<PeerList>((signal) => api.peers(signal), 3000);

    return (
        <section className="page">
            <h2>Wallet UI</h2>
            <p className="muted">
                Connected to node API: <code className="code">{env.nodeApiBaseUrl}</code>
            </p>

            <div className="grid">
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
                            <div className="row">
                                <span>Started</span>
                                <span className="value">{status.data.startedAt}</span>
                            </div>
                            <div className="row">
                                <span>Data Dir</span>
                                <span className="value">{status.data.dataDir}</span>
                            </div>
                        </div>
                    )}

                    {status.lastUpdated && <p className="tiny muted">Updated: {new Date(status.lastUpdated).toISOString()}</p>}
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

                    {peers.lastUpdated && <p className="tiny muted">Updated: {new Date(peers.lastUpdated).toISOString()}</p>}
                </div>
            </div>

            <div className="card">
                <h3>Next</h3>
                <p className="muted">
                    Next we’ll implement real wallet primitives in the UI (key creation, address display, local encryption), and a
                    safe node RPC surface for wallet operations.
                </p>
            </div>
        </section>
    );
}
