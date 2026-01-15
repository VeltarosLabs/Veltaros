import React, { useEffect, useRef, useState } from "react";
import { Link, NavLink, useLocation } from "react-router-dom";
import ThemeToggle from "./ThemeToggle";
import logoPng from "../assets/logo.png";

type Props = {
    theme: "light" | "dark";
    onToggleTheme: () => void;
};

const navLinks = [
    { to: "/", label: "Home" },
    { to: "/wallet", label: "Wallet" }
];

export default function Navbar({ theme, onToggleTheme }: Props): React.ReactElement {
    const [open, setOpen] = useState(false);
    const loc = useLocation();
    const menuRef = useRef<HTMLElement | null>(null);
    const toggleRef = useRef<HTMLButtonElement | null>(null);

    // Close menu on route change
    useEffect(() => {
        setOpen(false);
    }, [loc.pathname]);

    // Scroll lock while menu is open (mobile)
    useEffect(() => {
        if (!open) {
            document.body.classList.remove("noScroll");
            return;
        }
        document.body.classList.add("noScroll");
        return () => document.body.classList.remove("noScroll");
    }, [open]);

    // Close menu when clicking outside
    useEffect(() => {
        if (!open) return;

        const onDown = (e: MouseEvent | TouchEvent) => {
            const t = e.target as Node | null;
            if (!t) return;

            const menu = menuRef.current;
            const toggle = toggleRef.current;

            const insideMenu = menu ? menu.contains(t) : false;
            const insideToggle = toggle ? toggle.contains(t) : false;

            if (!insideMenu && !insideToggle) setOpen(false);
        };

        window.addEventListener("mousedown", onDown);
        window.addEventListener("touchstart", onDown);
        return () => {
            window.removeEventListener("mousedown", onDown);
            window.removeEventListener("touchstart", onDown);
        };
    }, [open]);

    // Close menu on Escape
    useEffect(() => {
        if (!open) return;

        const onKey = (e: KeyboardEvent) => {
            if (e.key === "Escape") setOpen(false);
        };
        window.addEventListener("keydown", onKey);
        return () => window.removeEventListener("keydown", onKey);
    }, [open]);

    return (
        <header className="nav">
            <div className="navInner">
                <Link to="/" className="brand" aria-label="Veltaros">
                    <img className="brandLogo" src={logoPng} alt="Veltaros logo" />
                    <span className="brandText">Veltaros</span>
                </Link>

                <div className="navRight">
                    <ThemeToggle theme={theme} onToggle={onToggleTheme} />

                    <button
                        ref={toggleRef}
                        type="button"
                        className="navToggle"
                        aria-label={open ? "Close menu" : "Open menu"}
                        aria-expanded={open}
                        onClick={() => setOpen((v) => !v)}
                    >
                        <span className={`bars ${open ? "open" : ""}`.trim()} />
                    </button>
                </div>

                <nav
                    ref={(el) => {
                        menuRef.current = el;
                    }}
                    className={`navLinks ${open ? "open" : ""}`.trim()}
                    aria-label="Primary navigation"
                >
                    {navLinks.map((l) => (
                        <NavLink key={l.to} to={l.to} end={l.to === "/"} className={({ isActive }) => (isActive ? "active" : "")}>
                            {l.label}
                        </NavLink>
                    ))}
                </nav>
            </div>
        </header>
    );
}
