import React, { useMemo, useState } from "react";
import { Link, NavLink, useLocation } from "react-router-dom";
import logoUrl from "../assets/veltaros-mark.svg";

const links = {
    github: "https://github.com/VeltarosLabs",
    x: "https://x.com/veltaros",
    reddit: "https://www.reddit.com/r/Veltaros/"
};

export default function Navbar(): React.ReactElement {
    const [open, setOpen] = useState(false);
    const loc = useLocation();

    useMemo(() => {
        setOpen(false);
        return null;
    }, [loc.pathname]);

    return (
        <header className="nav">
            <div className="navInner">
                <Link to="/" className="brand" aria-label="Veltaros home">
                    <img className="brandLogo" src={logoUrl} alt="Veltaros" />
                    <span className="brandText">Veltaros</span>
                </Link>

                <button
                    type="button"
                    className="navToggle"
                    aria-label={open ? "Close menu" : "Open menu"}
                    aria-expanded={open}
                    onClick={() => setOpen((v) => !v)}
                >
                    <span className="navToggleBars" />
                </button>

                <nav className={`navLinks ${open ? "open" : ""}`.trim()} aria-label="Primary navigation">
                    <NavLink to="/" end className={({ isActive }) => (isActive ? "active" : "")}>
                        Home
                    </NavLink>
                    <NavLink to="/wallet" className={({ isActive }) => (isActive ? "active" : "")}>
                        Wallet
                    </NavLink>

                    <div className="navDivider" />

                    <a href={links.github} target="_blank" rel="noreferrer noopener">
                        GitHub
                    </a>
                    <a href={links.x} target="_blank" rel="noreferrer noopener">
                        X
                    </a>
                    <a href={links.reddit} target="_blank" rel="noreferrer noopener">
                        Reddit
                    </a>
                </nav>
            </div>
        </header>
    );
}
