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

type LoadState<T> = { data: T | null; error: string | null; loading: boolean };

export default function Explorer(): React.ReactElement {
    const explorer = useMemo(() => new ExplorerClient(env.nodeApiBaseUrl), []);
    const [openHash, setOpenHash] = useState<string | null>(null);
    const [blockState, setBlockState] = useState<LoadState<StoredBlock>>({ data: null, error: null, loading: false });

    const tip = usePoll<TipInfo>((signal) => explorer.tip(signal), 2500);
    const blocks = usePoll<{ count: number; blocks: StoredBlockSummary[] }>((signal) => explorer.blocks(signal), 3000);

    const openBlock = async (hash: string) => {
        setOpenHash(hash);
        setBlockState({ data: null, error: null, loading: true });
        const ctrl = new AbortController();

        try {
            const b = await explorer.block(hash, ctrl.signal);
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
                                <div className="mono explorerLong">{tip.data.tipHash}</div>
                            </div>
                        </div>
                    )}
                </Card>

                <Card title="Recent Blocks" subtitle="Click a block to view details.">
                    {blocks.loading && <p className="muted">Loading…</p>}
                    {blocks.error && <p className="muted">{blocks.error}</p>}
                    {blocks.data && blocks.data.blocks.length === 0 && <p className="muted">No blocks yet.</p>}

                    {blocks.data && blocks.data.blocks.length > 0 && (
                        <div className="explorerBlocks">
                            {blocks.data.blocks.map((b) => (
                                <BlockCard key={b.hash} b={b} onOpen={openBlock} />
                            ))}
                        </div>
                    )}
                </Card>
            </div>

            <Modal open={Boolean(openHash)} title="Block Details" onClose={closeBlock}>
                {blockState.loading && <p className="muted">Loading block…</p>}
                {blockState.error && <p className="muted">{blockState.error}</p>}
                {blockState.data && <BlockDetails block={blockState.data} />}
            </Modal>
        </section>
    );
}
