import React from "react";
import { Link, NavLink } from "react-router-dom";

const links = {
    github: "https://github.com/VeltarosLabs",
    x: "https://x.com/veltaros",
    reddit: "https://www.reddit.com/r/Veltaros/"
};

export default function Navbar(): React.ReactElement {
    return (
        <header className="nav">
            <div className="nav-inner container">
                <Link to="/" className="brand" aria-label="Veltaros home">
                    Veltaros
                </Link>

                <nav className="nav-links" aria-label="Primary navigation">
                    <NavLink to="/" className={({ isActive }) => (isActive ? "active" : "")} end>
                        Home
                    </NavLink>
                    <NavLink to="/wallet" className={({ isActive }) => (isActive ? "active" : "")}>
                        Wallet
                    </NavLink>
                </nav>

                <div className="nav-social" aria-label="Social links">
                    <a href={links.github} target="_blank" rel="noreferrer noopener">
                        GitHub
                    </a>
                    <a href={links.x} target="_blank" rel="noreferrer noopener">
                        X
                    </a>
                    <a href={links.reddit} target="_blank" rel="noreferrer noopener">
                        Reddit
                    </a>
                </div>
            </div>
        </header>
    );
}
