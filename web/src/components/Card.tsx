import React from "react";

type Props = {
    title?: string;
    subtitle?: string;
    actions?: React.ReactNode;
    children: React.ReactNode;
    className?: string;
};

export default function Card({ title, subtitle, actions, children, className }: Props): React.ReactElement {
    return (
        <section className={`card ${className ?? ""}`.trim()}>
            {(title || subtitle || actions) && (
                <header className="cardHeader">
                    <div className="cardHeaderText">
                        {title && <h3 className="cardTitle">{title}</h3>}
                        {subtitle && <p className="cardSubtitle">{subtitle}</p>}
                    </div>
                    {actions && <div className="cardHeaderActions">{actions}</div>}
                </header>
            )}
            <div className="cardBody">{children}</div>
        </section>
    );
}
