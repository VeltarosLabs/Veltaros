import type { Health, NodeStatus, PeerList, VersionInfo } from "./types";

type Json = Record<string, unknown> | unknown[] | string | number | boolean | null;

export class VeltarosApiError extends Error {
    public readonly status: number;
    public readonly url: string;

    constructor(message: string, status: number, url: string) {
        super(message);
        this.name = "VeltarosApiError";
        this.status = status;
        this.url = url;
    }
}

export class VeltarosApiClient {
    private readonly baseUrl: string;

    constructor(baseUrl: string) {
        this.baseUrl = baseUrl.replace(/\/+$/, "");
    }

    async health(signal?: AbortSignal): Promise<Health> {
        return this.getJson<Health>("/healthz", signal);
    }

    async version(signal?: AbortSignal): Promise<VersionInfo> {
        return this.getJson<VersionInfo>("/version", signal);
    }

    async status(signal?: AbortSignal): Promise<NodeStatus> {
        return this.getJson<NodeStatus>("/status", signal);
    }

    async peers(signal?: AbortSignal): Promise<PeerList> {
        return this.getJson<PeerList>("/peers", signal);
    }

    private async getJson<T extends Json>(path: string, signal?: AbortSignal): Promise<T> {
        const url = `${this.baseUrl}${path}`;
        const res = await fetch(url, {
            method: "GET",
            headers: { Accept: "application/json" },
            signal
        });

        if (!res.ok) {
            const text = await safeReadText(res);
            throw new VeltarosApiError(`HTTP ${res.status} for ${path}${text ? `: ${text}` : ""}`, res.status, url);
        }

        // node API is strict JSON; if invalid, it will throw (that’s fine)
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
