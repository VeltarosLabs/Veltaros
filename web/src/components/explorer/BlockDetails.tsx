import React from "react";
import type { StoredBlock } from "../../api/explorerTypes";

function iso(ts: number): string {
    const d = new Date(ts * 1000);
    return isNaN(d.getTime()) ? "-" : d.toISOString();
}

export default function BlockDetails({ block }: { block: StoredBlock }): React.ReactElement {
    return (
        <div className="explorerDetails">
            <div className="explorerDetailsGrid">
                <div className="explorerKV">
                    <div className="muted tiny">Height</div>
                    <div className="mono">{block.height}</div>
                </div>
                <div className="explorerKV">
                    <div className="muted tiny">Time</div>
                    <div className="mono">{iso(block.timestamp)}</div>
                </div>
                <div className="explorerKV">
                    <div className="muted tiny">Tx count</div>
                    <div className="mono">{block.txCount}</div>
                </div>
                <div className="explorerKV">
                    <div className="muted tiny">Nonce</div>
                    <div className="mono">{block.block.header.nonce}</div>
                </div>
            </div>

            <div className="explorerSection">
                <div className="explorerSectionTitle">Block Hash</div>
                <div className="mono explorerLong">{block.hash}</div>
            </div>

            <div className="explorerSection">
                <div className="explorerSectionTitle">Previous Hash</div>
                <div className="mono explorerLong">{block.prevHash}</div>
            </div>

            <div className="explorerSection">
                <div className="explorerSectionTitle">Merkle Root</div>
                <div className="mono explorerLong">{block.merkleRoot}</div>
            </div>

            <div className="explorerSection">
                <div className="explorerSectionTitle">Transactions</div>

                {block.block.transactions.length === 0 ? (
                    <p className="muted">No transactions in this block.</p>
                ) : (
                    <div className="explorerTxList">
                        {block.block.transactions.map((tx) => (
                            <div key={tx.txId} className="explorerTx">
                                <div className="mono">{tx.txId}</div>
                                <div className="explorerTxRow">
                                    <span className="muted tiny mono">{tx.draft.from.slice(0, 16)}…</span>
                                    <span className="muted tiny">→</span>
                                    <span className="muted tiny mono">{tx.draft.to.slice(0, 16)}…</span>
                                </div>
                                <div className="explorerTxRow">
                                    <span className="muted tiny">Amount:</span>
                                    <span className="mono tiny">{tx.draft.amount}</span>
                                    <span className="muted tiny">Fee:</span>
                                    <span className="mono tiny">{tx.draft.fee}</span>
                                    <span className="muted tiny">Nonce:</span>
                                    <span className="mono tiny">{tx.draft.nonce}</span>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
}
