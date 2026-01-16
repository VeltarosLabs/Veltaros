import React from "react";
import type { StoredBlockSummary } from "../../api/explorerTypes";

type Props = {
    b: StoredBlockSummary;
    onOpen: (hash: string) => void;
};

function iso(ts: number): string {
    const d = new Date(ts * 1000);
    return isNaN(d.getTime()) ? "-" : d.toISOString();
}

export default function BlockCard({ b, onOpen }: Props): React.ReactElement {
    return (
        <button type="button" className="explorerBlock" onClick={() => onOpen(b.hash)}>
            <div className="explorerBlockTop">
                <div className="explorerBlockTitle">
                    <span className="explorerTag">Height</span>
                    <span className="mono">{b.height}</span>
                </div>
                <span className="explorerTag">{b.txCount} tx</span>
            </div>

            <div className="explorerHash mono">{b.hash}</div>

            <div className="explorerMeta">
                <div className="muted tiny">Time</div>
                <div className="mono tiny">{iso(b.timestamp)}</div>
            </div>

            <div className="explorerMeta">
                <div className="muted tiny">Prev</div>
                <div className="mono tiny">{b.prevHash.slice(0, 16)}…</div>
            </div>
        </button>
    );
}
