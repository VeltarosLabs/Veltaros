import React from "react";

type LinkItem = {
    label: string;
    href: string;
    icon: React.ReactNode;
};

function IconGitHub() {
    return (
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
            <path
                fill="currentColor"
                d="M12 2a10 10 0 0 0-3.16 19.49c.5.1.68-.22.68-.48v-1.7c-2.77.6-3.35-1.17-3.35-1.17-.45-1.14-1.1-1.45-1.1-1.45-.9-.62.07-.61.07-.61 1 .07 1.53 1.03 1.53 1.03.89 1.53 2.33 1.09 2.9.83.09-.65.35-1.09.63-1.34-2.21-.25-4.53-1.1-4.53-4.9 0-1.08.39-1.96 1.03-2.65-.1-.25-.45-1.27.1-2.65 0 0 .84-.27 2.75 1.02A9.5 9.5 0 0 1 12 6.8c.85 0 1.71.11 2.5.33 1.91-1.29 2.75-1.02 2.75-1.02.55 1.38.2 2.4.1 2.65.64.69 1.03 1.57 1.03 2.65 0 3.81-2.32 4.65-4.54 4.9.36.31.68.92.68 1.86v2.75c0 .26.18.59.69.48A10 10 0 0 0 12 2Z"
            />
        </svg>
    );
}

function IconX() {
    return (
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
            <path
                fill="currentColor"
                d="M18.9 2H22l-6.8 7.8L23.2 22h-6.6l-5.2-7.2L5.9 22H2.8l7.4-8.5L1 2h6.7l4.7 6.6L18.9 2Zm-1.2 18h1.8L6.2 3.9H4.3L17.7 20Z"
            />
        </svg>
    );
}

function IconReddit() {
    return (
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
            <path
                fill="currentColor"
                d="M20 12.2c0-1.1-.9-2-2-2-.5 0-1 .2-1.3.5-1.3-.9-3.1-1.5-5.1-1.6l1-4.6 3.2.7c0 1 .8 1.8 1.8 1.8 1 0 1.8-.8 1.8-1.8S19.6 3.4 18.6 3.4c-.7 0-1.3.4-1.6 1l-4-.9c-.5-.1-1 .2-1.1.7l-1.2 5.4c-2 .1-3.7.7-5.1 1.6-.4-.3-.8-.5-1.3-.5-1.1 0-2 .9-2 2 0 .8.5 1.5 1.2 1.8 0 .2-.1.5-.1.7 0 3 3.6 5.4 8 5.4s8-2.4 8-5.4c0-.2 0-.5-.1-.7.7-.3 1.2-1 1.2-1.8ZM8.7 15.7c-.7 0-1.2-.5-1.2-1.2s.5-1.2 1.2-1.2 1.2.5 1.2 1.2-.5 1.2-1.2 1.2Zm6.6 0c-.7 0-1.2-.5-1.2-1.2s.5-1.2 1.2-1.2 1.2.5 1.2 1.2-.5 1.2-1.2 1.2Zm-8 2.4c1.1 1 2.8 1.5 4.7 1.5s3.6-.6 4.7-1.5c.3-.3.3-.8 0-1.1-.3-.3-.8-.3-1.1 0-.8.7-2.1 1.1-3.6 1.1s-2.8-.4-3.6-1.1c-.3-.3-.8-.3-1.1 0-.3.3-.3.8 0 1.1Z"
            />
        </svg>
    );
}

const links: LinkItem[] = [
    { label: "GitHub", href: "https://github.com/VeltarosLabs", icon: <IconGitHub /> },
    { label: "X", href: "https://x.com/veltaros", icon: <IconX /> },
    { label: "Reddit", href: "https://www.reddit.com/r/Veltaros/", icon: <IconReddit /> }
];

export default function SocialLinks(): React.ReactElement {
    return (
        <div className="socialRow" aria-label="Social links">
            {links.map((l) => (
                <a
                    key={l.label}
                    className="socialLink"
                    href={l.href}
                    target="_blank"
                    rel="noreferrer noopener"
                    aria-label={l.label}
                >
                    {l.icon}
                    <span className="socialLabel">{l.label}</span>
                </a>
            ))}
        </div>
    );
}
