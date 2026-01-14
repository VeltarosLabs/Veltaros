import React from "react";

type LinkItem = {
    label: string;
    href: string;
    icon: React.ReactNode;
};

function IconInstagram() {
    return (
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
            <path
                fill="currentColor"
                d="M7 2h10a5 5 0 0 1 5 5v10a5 5 0 0 1-5 5H7a5 5 0 0 1-5-5V7a5 5 0 0 1 5-5Zm10 2H7a3 3 0 0 0-3 3v10a3 3 0 0 0 3 3h10a3 3 0 0 0 3-3V7a3 3 0 0 0-3-3Zm-5 4a6 6 0 1 1 0 12 6 6 0 0 1 0-12Zm0 2a4 4 0 1 0 0 8 4 4 0 0 0 0-8Zm6.5-2.25a1.25 1.25 0 1 1 0 2.5 1.25 1.25 0 0 1 0-2.5Z"
            />
        </svg>
    );
}

function IconFacebook() {
    return (
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
            <path
                fill="currentColor"
                d="M13 22v-8h3l1-4h-4V7.5c0-1 .3-1.5 1.7-1.5H17V2.1c-.6-.1-1.9-.1-3.4-.1C10.9 2 9 3.6 9 6.7V10H6v4h3v8h4Z"
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

function IconYouTube() {
    return (
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
            <path
                fill="currentColor"
                d="M21.6 7.2a3 3 0 0 0-2.1-2.1C17.6 4.6 12 4.6 12 4.6s-5.6 0-7.5.5A3 3 0 0 0 2.4 7.2 31.4 31.4 0 0 0 2 12a31.4 31.4 0 0 0 .4 4.8 3 3 0 0 0 2.1 2.1c1.9.5 7.5.5 7.5.5s5.6 0 7.5-.5a3 3 0 0 0 2.1-2.1A31.4 31.4 0 0 0 22 12a31.4 31.4 0 0 0-.4-4.8ZM10 15.5v-7l6 3.5-6 3.5Z"
            />
        </svg>
    );
}

function IconTikTok() {
    return (
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
            <path
                fill="currentColor"
                d="M16.6 5.6c-1-.6-1.7-1.5-2-2.6h-2.7v12.1a2.4 2.4 0 1 1-2-2.4c.2 0 .4 0 .6.1V9.9c-.2 0-.4-.1-.6-.1A5.2 5.2 0 1 0 13 15.1V8.3c1.1.8 2.4 1.2 3.8 1.2V6.8c-.7 0-1.4-.2-2.2-.7Z"
            />
        </svg>
    );
}

const links: LinkItem[] = [
    { label: "Instagram", href: "https://instagram.com", icon: <IconInstagram /> },
    { label: "Facebook", href: "https://facebook.com", icon: <IconFacebook /> },
    { label: "Twitter/X", href: "https://twitter.com", icon: <IconX /> },
    { label: "YouTube", href: "https://youtube.com", icon: <IconYouTube /> },
    { label: "TikTok", href: "https://tiktok.com", icon: <IconTikTok /> }
];

export default function SocialLinks(): React.ReactElement {
    return (
        <div className="socialRow" aria-label="Social links">
            {links.map((l) => (
                <a key={l.label} className="socialLink" href={l.href} target="_blank" rel="noreferrer noopener" aria-label={l.label}>
                    {l.icon}
                    <span className="socialLabel">{l.label}</span>
                </a>
            ))}
        </div>
    );
}
