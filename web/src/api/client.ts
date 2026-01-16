import type { Health, NodeStatus, PeerList, VersionInfo } from "./types";
import type { MempoolResponse } from "./mempoolTypes";
import type { AccountInfo } from "./accountTypes";
import type { SignedTx } from "../tx/types";

type Json = Record<string, unknown> | unknown[] | string | number | boolean | null;

export type TxValidateOk = {
    ok: true;
    txId: string;
    from: string;
    lastNonce: number;
    expectedNonce: number;
    mempoolHas: boolean;
};

export type TxValidateErr = {
    ok: false;
    error: string;
    lastNonce?: number;
    expectedNonce?: number;
};

export type TxBroadcastOk = {
    ok: true;
    txId: string;
    note?: string;
};

export type TxBroadcastErr = {
    ok: false;
    error: string;
    lastNonce?: number;
    expectedNonce?: number;
};

export type TxValidateResponse = TxValidateOk | TxValidateErr;
export type TxBroadcastResponse = TxBroadcastOk | TxBroadcastErr;

export type ProduceBlockResponse =
    | { ok: true; applied: number; failed: number; height: number }
    | { ok?: false; error: string };

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
    private readonly apiKey: string;

    constructor(baseUrl: string, apiKey?: string) {
        this.baseUrl = baseUrl.replace(/\/+$/, "");
        this.apiKey = (apiKey ?? "").trim();
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

    async mempool(signal?: AbortSignal): Promise<MempoolResponse> {
        return this.getJson<MempoolResponse>("/mempool", signal);
    }

    async account(address: string, signal?: AbortSignal): Promise<AccountInfo> {
        const safe = encodeURIComponent(address.trim());
        return this.getJson<AccountInfo>(`/account/${safe}`, signal);
    }

    async txValidate(tx: SignedTx, signal?: AbortSignal): Promise<TxValidateResponse> {
        return this.postJson<TxValidateResponse>("/tx/validate", tx, signal, true);
    }

    async txBroadcast(tx: SignedTx, signal?: AbortSignal): Promise<TxBroadcastResponse> {
        return this.postJson<TxBroadcastResponse>("/tx/broadcast", tx, signal, true);
    }

    async produceBlock(signal?: AbortSignal): Promise<ProduceBlockResponse> {
        return this.postJson<ProduceBlockResponse>("/dev/produce-block", {}, signal, true);
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

        return (await res.json()) as T;
    }

    private async postJson<T extends Json>(
        path: string,
        body: unknown,
        signal?: AbortSignal,
        includeApiKey?: boolean
    ): Promise<T> {
        const url = `${this.baseUrl}${path}`;

        const headers: Record<string, string> = {
            Accept: "application/json",
            "Content-Type": "application/json"
        };

        if (includeApiKey && this.apiKey) {
            headers["X-API-Key"] = this.apiKey;
        }

        const res = await fetch(url, {
            method: "POST",
            headers,
            body: JSON.stringify(body),
            signal
        });

        const text = await res.text();
        const json = safeParseJson(text);

        if (!res.ok) {
            if (json) return json as T;
            throw new VeltarosApiError(`HTTP ${res.status} for ${path}${text ? `: ${text.slice(0, 200)}` : ""}`, res.status, url);
        }

        if (json) return json as T;
        return {} as T;
    }
}

function safeParseJson(text: string): unknown | null {
    const t = text.trim();
    if (!t) return null;
    try {
        return JSON.parse(t);
    } catch {
        return null;
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
