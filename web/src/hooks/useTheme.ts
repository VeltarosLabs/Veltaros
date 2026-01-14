import { useCallback, useEffect, useMemo, useState } from "react";

type Theme = "light" | "dark";

const STORAGE_KEY = "veltaros.theme";

function getSystemTheme(): Theme {
    if (typeof window === "undefined") return "dark";
    return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function applyTheme(theme: Theme) {
    document.documentElement.dataset.theme = theme;
}

export function useTheme() {
    const [theme, setTheme] = useState<Theme>(() => {
        const saved = typeof window !== "undefined" ? localStorage.getItem(STORAGE_KEY) : null;
        if (saved === "light" || saved === "dark") return saved;
        return getSystemTheme();
    });

    useEffect(() => {
        applyTheme(theme);
        localStorage.setItem(STORAGE_KEY, theme);
    }, [theme]);

    useEffect(() => {
        // If user hasn't set a preference yet, track system changes
        const saved = localStorage.getItem(STORAGE_KEY);
        if (saved === "light" || saved === "dark") return;

        const mq = window.matchMedia("(prefers-color-scheme: dark)");
        const onChange = () => setTheme(mq.matches ? "dark" : "light");

        mq.addEventListener?.("change", onChange);
        return () => mq.removeEventListener?.("change", onChange);
    }, []);

    const toggle = useCallback(() => {
        setTheme((t) => (t === "dark" ? "light" : "dark"));
    }, []);

    const api = useMemo(() => ({ theme, setTheme, toggle }), [theme, toggle]);
    return api;
}
