import type { SignedTx } from "./types";

const KEY = "veltaros.tx.history.v1";

export type TxHistoryItem = {
    id: string;
    createdAt: string;
    status: "drafted" | "validated" | "broadcast" | "error";
    note?: string;
    tx: SignedTx;
};

function safeParse(raw: string | null): TxHistoryItem[] {
    if (!raw) return [];
    try {
        const arr = JSON.parse(raw) as TxHistoryItem[];
        if (!Array.isArray(arr)) return [];
        return arr.filter((x) => x && typeof x.id === "string" && x.tx && typeof x.tx === "object");
    } catch {
        return [];
    }
}

export function loadHistory(): TxHistoryItem[] {
    return safeParse(localStorage.getItem(KEY));
}

export function saveHistory(items: TxHistoryItem[]): void {
    localStorage.setItem(KEY, JSON.stringify(items.slice(0, 200)));
}

export function upsertHistory(item: TxHistoryItem): void {
    const all = loadHistory();
    const idx = all.findIndex((x) => x.id === item.id);
    if (idx >= 0) all[idx] = item;
    else all.unshift(item);
    saveHistory(all);
}

export function clearHistory(): void {
    localStorage.removeItem(KEY);
}
