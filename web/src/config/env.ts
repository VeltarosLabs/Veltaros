export const env = {
    nodeApiBaseUrl: (import.meta.env.VITE_VELTAROS_NODE_API as string | undefined)?.trim() || "http://127.0.0.1:8080",
    apiKey: (import.meta.env.VITE_VELTAROS_API_KEY as string | undefined)?.trim() || ""
} as const;
