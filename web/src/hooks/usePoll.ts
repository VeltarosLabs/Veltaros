import { useEffect, useRef, useState } from "react";

type PollState<T> = {
    data: T | null;
    error: string | null;
    loading: boolean;
    lastUpdated: number | null;
};

export function usePoll<T>(fn: (signal: AbortSignal) => Promise<T>, intervalMs: number) {
    const [state, setState] = useState<PollState<T>>({
        data: null,
        error: null,
        loading: true,
        lastUpdated: null
    });

    const backoffRef = useRef(0);

    useEffect(() => {
        let mounted = true;
        let timer: number | undefined;
        const controller = new AbortController();

        const tick = async () => {
            if (!mounted) return;

            setState((s) => ({ ...s, loading: s.data === null }));
            try {
                const data = await fn(controller.signal);
                if (!mounted) return;

                backoffRef.current = 0;
                setState({ data, error: null, loading: false, lastUpdated: Date.now() });

                // normal interval
                timer = window.setTimeout(tick, intervalMs);
            } catch (e) {
                if (!mounted) return;
                if (controller.signal.aborted) return;

                const msg = e instanceof Error ? e.message : "Unknown error";
                setState((s) => ({ ...s, error: msg, loading: false }));

                // backoff: 1s -> 2s -> 4s -> 8s (cap), then continue
                backoffRef.current = Math.min(backoffRef.current === 0 ? 1000 : backoffRef.current * 2, 8000);
                timer = window.setTimeout(tick, backoffRef.current);
            }
        };

        tick();

        return () => {
            mounted = false;
            controller.abort();
            if (timer) window.clearTimeout(timer);
        };
    }, [fn, intervalMs]);

    return state;
}
