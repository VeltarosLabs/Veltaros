import React from "react";

export type TabItem<T extends string> = {
    key: T;
    label: string;
    badge?: string;
};

type Props<T extends string> = {
    items: Array<TabItem<T>>;
    value: T;
    onChange: (v: T) => void;
};

export default function Tabs<T extends string>({ items, value, onChange }: Props<T>): React.ReactElement {
    return (
        <div className="tabs" role="tablist" aria-label="Sections">
            {items.map((t) => {
                const active = t.key === value;
                return (
                    <button
                        key={t.key}
                        type="button"
                        className={`tab ${active ? "active" : ""}`.trim()}
                        role="tab"
                        aria-selected={active}
                        onClick={() => onChange(t.key)}
                    >
                        <span>{t.label}</span>
                        {t.badge && <span className="tabBadge">{t.badge}</span>}
                    </button>
                );
            })}
        </div>
    );
}
