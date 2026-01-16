import React, { useMemo } from "react";
import { env } from "../config/env";
import { ExplorerClient } from "../api/explorerClient";
import type { TipInfo } from "../api/explorerTypes";
import { usePoll } from "../hooks/usePoll";

function shortHash(h: string): string {
    if (!h) return "-";
    if (h.length <= 18) return h;
    return `${h.slice(0, 10)}…${h.slice(-8)}`;
}

export default function TipPreview(): React.ReactElement {
    const client = useMemo(() => new ExplorerClient(env.nodeApiBaseUrl), []);
    const tip = usePoll<TipInfo>((signal) => client.tip(signal), 3000);

    if (tip.loading) {
        return <div className="tipBox muted">Loading tip…</div>;
    }

    if (tip.error) {
        return <div className="tipBox muted">Tip unavailable</div>;
    }

    if (!tip.data) {
        return <div className="tipBox muted">Tip unavailable</div>;
    }

    return (
        <div className="tipBox">
            <div className="tipRow">
                <span className="muted tiny">Height</span>
                <span className="mono">{tip.data.height}</span>
            </div>

            <div className="tipRow">
                <span className="muted tiny">Tip</span>
                <span className="mono">{shortHash(tip.data.tipHash)}</span>
            </div>
        </div>
    );
}
