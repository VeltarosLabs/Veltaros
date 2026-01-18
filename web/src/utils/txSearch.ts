import type { StoredBlockSummary } from "../api/explorerTypes";

export function normalizeTxId(txId: string): string {
    return txId.trim().toLowerCase();
}

export function txInBlockSummary(
    txId: string,
    block: StoredBlockSummary & { txIds?: string[] }
): boolean {
    if (!block.txIds || block.txIds.length === 0) return false;
    const needle = normalizeTxId(txId);
    return block.txIds.some((id) => id.toLowerCase() === needle);
}
