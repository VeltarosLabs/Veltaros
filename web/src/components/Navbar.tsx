import React, { useEffect, useState } from "react";
import { Link, NavLink } from "react-router-dom";
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

    useEffect(() => {
        const onResize = () => {
            if (window.innerWidth > 720) setOpen(false);
        };
        window.addEventListener("resize", onResize);
        return () => window.removeEventListener("resize", onResize);
    }, []);

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
                        type="button"
                        className="navToggle"
                        aria-label={open ? "Close menu" : "Open menu"}
                        aria-expanded={open}
                        onClick={() => setOpen((v) => !v)}
                    >
                        <span className={`bars ${open ? "open" : ""}`.trim()} />
                    </button>
                </div>

                <nav className={`navLinks ${open ? "open" : ""}`.trim()} aria-label="Primary navigation">
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
