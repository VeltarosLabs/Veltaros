import type { BlocksResponse, StoredBlock, TipInfo } from "./explorerTypes";
import { VeltarosApiError } from "./client";

export class ExplorerClient {
    private readonly baseUrl: string;

    constructor(baseUrl: string) {
        this.baseUrl = baseUrl.replace(/\/+$/, "");
    }

    async tip(signal?: AbortSignal): Promise<TipInfo> {
        return this.getJson<TipInfo>("/tip", signal);
    }

    async blocks(signal?: AbortSignal): Promise<BlocksResponse> {
        return this.getJson<BlocksResponse>("/blocks", signal);
    }

    async block(hash: string, signal?: AbortSignal): Promise<StoredBlock> {
        const safe = encodeURIComponent(hash.trim());
        return this.getJson<StoredBlock>(`/block/${safe}`, signal);
    }

    private async getJson<T>(path: string, signal?: AbortSignal): Promise<T> {
        const url = `${this.baseUrl}${path}`;
        const res = await fetch(url, { method: "GET", headers: { Accept: "application/json" }, signal });

        if (!res.ok) {
            const text = await safeReadText(res);
            throw new VeltarosApiError(`HTTP ${res.status} for ${path}${text ? `: ${text}` : ""}`, res.status, url);
        }
        return (await res.json()) as T;
    }
}

async function safeReadText(res: Response): Promise<string> {
    try {
        const t = await res.text();
        return t.trim().slice(0, 400);
    } catch {
        return "";
    }
}
