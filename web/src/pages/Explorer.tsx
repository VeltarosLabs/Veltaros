import React, { useMemo, useState } from "react";
import { env } from "../config/env";
import Card from "../components/Card";
import Modal from "../components/Modal";
import { ExplorerClient } from "../api/explorerClient";
import type {
    StoredBlock,
    StoredBlockSummary,
    TipInfo,
    NodeStatusLite
} from "../api/explorerTypes";
import BlockDetails from "../components/explorer/BlockDetails";
import "../styles/explorer.css";
import { usePoll } from "../hooks/usePoll";
import { copyText } from "../utils/clipboard";
import { normalizeTxId } from "../utils/txSearch";
import { Link } from "react-router-dom";

type LoadState<T> = { data: T | null; error: string | null; loading: boolean };

function shortHash(h: string): string {
    if (!h) return "-";
    if (h.length <= 20) return h;
    return `${h.slice(0, 12)}…${h.slice(-8)}`;
}

function iso(ts: number): string {
    const d = new Date(ts * 1000);
    return isNaN(d.getTime()) ? "-" : d.toISOString();
}

export default function Explorer(): React.ReactElement {
    const explorer = useMemo(() => new ExplorerClient(env.nodeApiBaseUrl), []);
    const [openHash, setOpenHash] = useState<string | null>(null);
    const [blockState, setBlockState] = useState<LoadState<StoredBlock>>({
        data: null,
        error: null,
        loading: false
    });

    const [blockSearch, setBlockSearch] = useState("");
    const [txSearch, setTxSearch] = useState("");
    const [notice, setNotice] = useState<string | null>(null);
    const [visibleCount, setVisibleCount] = useState(25);

    const tip = usePoll<TipInfo>((s) => explorer.tip(s), 2500);
    const blocks = usePoll<{ count: number; blocks: StoredBlockSummary[] }>(
        (s) => explorer.blocks(s),
        3000
    );

    const statusLite = usePoll<NodeStatusLite>(async (signal) => {
        const url = `${env.nodeApiBaseUrl.replace(/\/+$/, "")}/status`;
        const res = await fetch(url, {
            method: "GET",
            headers: { Accept: "application/json" },
            signal
        });
        if (!res.ok) throw new Error("Status unavailable");
        return (await res.json()) as NodeStatusLite;
    }, 3000);

    const devMode = Boolean(statusLite.data?.devMode);
    const allBlocks = blocks.data?.blocks ?? [];
    const shownBlocks = allBlocks.slice(
        Math.max(0, allBlocks.length - visibleCount)
    );

    const openBlock = async (hash: string) => {
        const h = hash.trim();
        if (!h) return;

        setOpenHash(h);
        setBlockState({ data: null, error: null, loading: true });

        try {
            const b = await explorer.block(h);
            setBlockState({ data: b, error: null, loading: false });
        } catch (e) {
            setBlockState({
                data: null,
                error: e instanceof Error ? e.message : "Failed to load block",
                loading: false
            });
        }
    };

    const closeBlock = () => {
        setOpenHash(null);
        setBlockState({ data: null, error: null, loading: false });
    };

    const onCopy = async (text: string) => {
        try {
            await copyText(text);
            setNotice("Copied");
            window.setTimeout(() => setNotice(null), 1400);
        } catch {
            setNotice("Copy failed");
            window.setTimeout(() => setNotice(null), 1600);
        }
    };

    const onBlockSearch = async () => {
        if (!blockSearch.trim()) return;
        await openBlock(blockSearch);
    };

    const onTxSearch = async () => {
        const txId = normalizeTxId(txSearch);
        if (!txId) {
            setNotice("Enter a transaction ID");
            return;
        }

        // brute-force recent blocks (safe + read-only)
        for (let i = allBlocks.length - 1; i >= 0; i--) {
            try {
                const b = await explorer.block(allBlocks[i].hash);
                const found = b.block.transactions.some(
                    (tx) => tx.txId.toLowerCase() === txId
                );
                if (found) {
                    setNotice("Transaction found");
                    await openBlock(b.hash);
                    return;
                }
            } catch {
                continue;
            }
        }

        setNotice("Transaction not found in recent blocks");
        window.setTimeout(() => setNotice(null), 2200);
    };

    return (
        <section className="page">
            <div className="pageTop">
                <div>
                    <h2 className="pageTitle">Explorer</h2>
                    <p className="muted">
                        Live chain view: <span className="mono">{env.nodeApiBaseUrl}</span>
                    </p>
                </div>
            </div>

            {notice && <div className="alert">{notice}</div>}

            {devMode && (
                <div className="alert explorerDev">
                    <strong>Dev mode enabled.</strong>
                    <div className="muted">
                        Produce blocks from the Wallet to confirm transactions.
                    </div>
                    <Link to="/wallet" className="btn small primary">
                        Go to Wallet
                    </Link>
                </div>
            )}

            <div className="explorerGrid">
                <Card title="Tip">
                    {tip.data && (
                        <div className="explorerTip">
                            <div>
                                <div className="muted tiny">Height</div>
                                <div className="mono">{tip.data.height}</div>
                            </div>
                            <div>
                                <div className="muted tiny">Hash</div>
                                <div className="mono">{shortHash(tip.data.tipHash)}</div>
                            </div>
                        </div>
                    )}
                </Card>

                <Card title="Search">
                    <label className="label">
                        Block hash
                        <input
                            className="input mono"
                            value={blockSearch}
                            onChange={(e) => setBlockSearch(e.target.value)}
                        />
                    </label>

                    <button className="btn primary small" onClick={onBlockSearch}>
                        Open block
                    </button>

                    <label className="label" style={{ marginTop: "0.75rem" }}>
                        Transaction ID
                        <input
                            className="input mono"
                            value={txSearch}
                            onChange={(e) => setTxSearch(e.target.value)}
                        />
                    </label>

                    <button className="btn small" onClick={onTxSearch}>
                        Find transaction
                    </button>
                </Card>

                <Card title="Blocks">
                    <div className="explorerTableWrap">
                        <table className="explorerTable">
                            <thead>
                                <tr>
                                    <th>Height</th>
                                    <th>Hash</th>
                                    <th>Tx</th>
                                    <th>Time</th>
                                    <th />
                                </tr>
                            </thead>
                            <tbody>
                                {shownBlocks
                                    .slice()
                                    .reverse()
                                    .map((b) => (
                                        <tr
                                            key={b.hash}
                                            className="explorerRowClickable"
                                            onClick={() => openBlock(b.hash)}
                                        >
                                            <td className="mono">{b.height}</td>
                                            <td className="mono">{shortHash(b.hash)}</td>
                                            <td className="mono">{b.txCount}</td>
                                            <td className="mono tiny">{iso(b.timestamp)}</td>
                                            <td
                                                onClick={(e) => e.stopPropagation()}
                                                style={{ textAlign: "right" }}
                                            >
                                                <button
                                                    className="btn small"
                                                    onClick={() => onCopy(b.hash)}
                                                >
                                                    Copy
                                                </button>
                                            </td>
                                        </tr>
                                    ))}
                            </tbody>
                        </table>
                    </div>

                    {visibleCount < allBlocks.length && (
                        <button
                            className="btn small"
                            style={{ marginTop: "0.75rem" }}
                            onClick={() => setVisibleCount((n) => n + 25)}
                        >
                            Load more
                        </button>
                    )}
                </Card>
            </div>

            <Modal open={Boolean(openHash)} title="Block Details" onClose={closeBlock}>
                {blockState.loading && <p className="muted">Loading…</p>}
                {blockState.error && <p className="muted">{blockState.error}</p>}
                {blockState.data && <BlockDetails block={blockState.data} />}
            </Modal>
        </section>
    );
}
