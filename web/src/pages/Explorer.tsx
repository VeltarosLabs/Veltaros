import React, { useMemo, useState } from "react";
import { env } from "../config/env";
import Card from "../components/Card";
import Modal from "../components/Modal";
import { ExplorerClient } from "../api/explorerClient";
import type { StoredBlock, StoredBlockSummary, TipInfo } from "../api/explorerTypes";
import BlockCard from "../components/explorer/BlockCard";
import BlockDetails from "../components/explorer/BlockDetails";
import "../styles/explorer.css";
import { usePoll } from "../hooks/usePoll";
import { copyText } from "../utils/clipboard";

type LoadState<T> = { data: T | null; error: string | null; loading: boolean };

function shortHash(h: string): string {
    if (!h) return "-";
    if (h.length <= 20) return h;
    return `${h.slice(0, 12)}…${h.slice(-8)}`;
}

export default function Explorer(): React.ReactElement {
    const explorer = useMemo(() => new ExplorerClient(env.nodeApiBaseUrl), []);
    const [openHash, setOpenHash] = useState<string | null>(null);
    const [blockState, setBlockState] = useState<LoadState<StoredBlock>>({ data: null, error: null, loading: false });

    const [search, setSearch] = useState("");
    const [notice, setNotice] = useState<string | null>(null);

    const tip = usePoll<TipInfo>((signal) => explorer.tip(signal), 2500);
    const blocks = usePoll<{ count: number; blocks: StoredBlockSummary[] }>((signal) => explorer.blocks(signal), 3000);

    const openBlock = async (hash: string) => {
        const h = hash.trim();
        if (!h) return;

        setOpenHash(h);
        setBlockState({ data: null, error: null, loading: true });

        const ctrl = new AbortController();
        try {
            const b = await explorer.block(h, ctrl.signal);
            setBlockState({ data: b, error: null, loading: false });
        } catch (e) {
            const msg = e instanceof Error ? e.message : "Failed to load block";
            setBlockState({ data: null, error: msg, loading: false });
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

    const onSearch = async () => {
        const q = search.trim();
        if (!q) {
            setNotice("Enter a block hash");
            window.setTimeout(() => setNotice(null), 1600);
            return;
        }
        await openBlock(q);
    };

    const hasBlocks = Boolean(blocks.data && blocks.data.blocks && blocks.data.blocks.length > 0);

    return (
        <section className="page">
            <div className="pageTop">
                <div>
                    <h2 className="pageTitle">Explorer</h2>
                    <p className="muted">
                        Live chain view from your node: <span className="mono">{env.nodeApiBaseUrl}</span>
                    </p>
                </div>
            </div>

            {notice && <div className="alert">{notice}</div>}

            <div className="explorerGrid">
                <Card title="Tip" subtitle="Current chain tip reported by the node.">
                    {tip.loading && <p className="muted">Loading…</p>}
                    {tip.error && <p className="muted">{tip.error}</p>}

                    {tip.data && (
                        <div className="explorerTip">
                            <div className="explorerKV">
                                <div className="muted tiny">Height</div>
                                <div className="mono">{tip.data.height}</div>
                            </div>

                            <div className="explorerKV">
                                <div className="muted tiny">Tip Hash</div>
                                <div className="explorerRow">
                                    <div className="mono explorerLong">{tip.data.tipHash}</div>
                                    <button className="btn small" type="button" onClick={() => void onCopy(tip.data.tipHash)}>
                                        Copy
                                    </button>
                                </div>
                                <div className="muted tiny" style={{ marginTop: "0.35rem" }}>
                                    {shortHash(tip.data.tipHash)}
                                </div>
                            </div>
                        </div>
                    )}
                </Card>

                <Card title="Search" subtitle="Paste a block hash to open details.">
                    <label className="label">
                        Block hash
                        <input
                            className="input mono"
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            placeholder="block hash"
                        />
                    </label>

                    <div className="rowBtns">
                        <button className="btn primary" type="button" onClick={() => void onSearch()}>
                            Open block
                        </button>
                        <button
                            className="btn"
                            type="button"
                            onClick={() => {
                                setSearch("");
                                setNotice(null);
                            }}
                        >
                            Clear
                        </button>
                    </div>

                    {!hasBlocks && (
                        <div className="explorerHint">
                            <p className="muted" style={{ margin: 0 }}>
                                No blocks yet. In dev mode, produce a block to generate your first history.
                            </p>
                        </div>
                    )}
                </Card>

                <Card title="Recent Blocks" subtitle="Click a block to view details.">
                    {blocks.loading && <p className="muted">Loading…</p>}
                    {blocks.error && <p className="muted">{blocks.error}</p>}

                    {blocks.data && blocks.data.blocks.length === 0 && (
                        <p className="muted">No blocks have been stored yet.</p>
                    )}

                    {blocks.data && blocks.data.blocks.length > 0 && (
                        <div className="explorerBlocks">
                            {blocks.data.blocks.map((b) => (
                                <div key={b.hash} className="explorerBlockWrap">
                                    <BlockCard b={b} onOpen={openBlock} />
                                    <div className="explorerBlockActions">
                                        <button className="btn small" type="button" onClick={() => void onCopy(b.hash)}>
                                            Copy hash
                                        </button>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </Card>
            </div>

            <Modal open={Boolean(openHash)} title="Block Details" onClose={closeBlock}>
                {blockState.loading && <p className="muted">Loading block…</p>}
                {blockState.error && <p className="muted">{blockState.error}</p>}
                {blockState.data && (
                    <>
                        <div className="rowBtns" style={{ marginTop: 0 }}>
                            <button className="btn small" type="button" onClick={() => void onCopy(blockState.data.hash)}>
                                Copy block hash
                            </button>
                            <button className="btn small" type="button" onClick={() => void onCopy(blockState.data.prevHash)}>
                                Copy prev hash
                            </button>
                            <button className="btn small" type="button" onClick={() => void onCopy(blockState.data.merkleRoot)}>
                                Copy merkle root
                            </button>
                        </div>

                        <BlockDetails block={blockState.data} />
                    </>
                )}
            </Modal>
        </section>
    );
}
