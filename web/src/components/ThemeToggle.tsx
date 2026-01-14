import React from "react";

type Props = {
    theme: "light" | "dark";
    onToggle: () => void;
};

function SunIcon(): React.ReactElement {
    return (
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
            <path
                d="M12 18a6 6 0 1 1 0-12 6 6 0 0 1 0 12Zm0-16a1 1 0 0 1 1 1v1a1 1 0 1 1-2 0V3a1 1 0 0 1 1-1Zm0 18a1 1 0 0 1 1 1v1a1 1 0 1 1-2 0v-1a1 1 0 0 1 1-1ZM4 11a1 1 0 0 1 0 2H3a1 1 0 1 1 0-2h1Zm18 0a1 1 0 0 1 0 2h-1a1 1 0 1 1 0-2h1ZM5.64 5.64a1 1 0 0 1 1.42 0l.7.7A1 1 0 0 1 6.34 7.76l-.7-.7a1 1 0 0 1 0-1.42Zm12.02 12.02a1 1 0 0 1 1.42 0l.7.7a1 1 0 1 1-1.42 1.42l-.7-.7a1 1 0 0 1 0-1.42ZM18.36 5.64a1 1 0 0 1 0 1.42l-.7.7a1 1 0 0 1-1.42-1.42l.7-.7a1 1 0 0 1 1.42 0ZM7.76 16.24a1 1 0 0 1 0 1.42l-.7.7a1 1 0 1 1-1.42-1.42l.7-.7a1 1 0 0 1 1.42 0Z"
                fill="currentColor"
            />
        </svg>
    );
}

function MoonIcon(): React.ReactElement {
    return (
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
            <path
                d="M21 14.5A8.5 8.5 0 0 1 9.5 3a7 7 0 1 0 11.5 11.5Z"
                fill="currentColor"
            />
        </svg>
    );
}

export default function ThemeToggle({ theme, onToggle }: Props): React.ReactElement {
    const isDark = theme === "dark";
    return (
        <button type="button" className="iconBtn" onClick={onToggle} aria-label="Toggle theme">
            {isDark ? <SunIcon /> : <MoonIcon />}
            <span className="iconBtnLabel">{isDark ? "Light" : "Dark"}</span>
        </button>
    );
}
