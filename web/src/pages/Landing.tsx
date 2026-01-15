import React from "react";
import { Link } from "react-router-dom";
import logoPng from "../assets/logo.png";
import heroImg from "../assets/hero.jpg";
import "../styles/landing.css";

export default function Landing(): React.ReactElement {
    return (
        <main className="landing">
            {/* HERO */}
            <section className="hero" style={{ backgroundImage: `url(${heroImg})` }}>
                <div className="heroOverlay" />

                <div className="heroContent">
                    <img src={logoPng} alt="Veltaros logo" className="heroLogo" />

                    <h1 className="heroTitle">Veltaros</h1>
                    <p className="heroSubtitle">
                        A decentralized digital currency focused on security, transparency, and long-term sustainability.
                    </p>

                    <div className="heroActions">
                        <Link to="/wallet" className="btn primary">
                            Open Wallet
                        </Link>
                        <a href="#vision" className="btn ghost">
                            Learn More
                        </a>
                    </div>
                </div>
            </section>

            {/* VISION */}
            <section id="vision" className="section">
                <header className="sectionHeader">
                    <h2>Vision</h2>
                    <p>
                        Veltaros is built as a complete blockchain ecosystem — designed for real-world use with clean architecture
                        and strict validation rules.
                    </p>
                </header>

                <div className="cardGrid">
                    <div className="card">
                        <h3>Decentralized by Design</h3>
                        <p>
                            A peer-to-peer network with independent nodes, transparent consensus, and no central authority.
                        </p>
                    </div>

                    <div className="card">
                        <h3>Security First</h3>
                        <p>
                            Modern cryptography, signed transactions, replay protection, and protocol-level validation to reduce risk.
                        </p>
                    </div>

                    <div className="card">
                        <h3>Built for the Long Term</h3>
                        <p>
                            Maintainable components, predictable behavior, and a focus on correctness over shortcuts.
                        </p>
                    </div>
                </div>
            </section>

            {/* TECHNOLOGY */}
            <section className="section alt">
                <header className="sectionHeader">
                    <h2>Technology</h2>
                    <p>A full-stack blockchain system built with performance, clarity, and safety in mind.</p>
                </header>

                <div className="cardGrid">
                    <div className="card">
                        <h3>Core Engine</h3>
                        <p>
                            Implemented in Go for performance and concurrency. Includes networking, transaction validation, and state
                            foundations.
                        </p>
                    </div>

                    <div className="card">
                        <h3>Wallet Experience</h3>
                        <p>
                            A modern interface designed for desktop and mobile. Local encrypted vault, address tools, and clean send
                            flows.
                        </p>
                    </div>

                    <div className="card">
                        <h3>Network Visibility</h3>
                        <p>
                            View peer connections and mempool activity clearly, with controls that stay readable in light and dark
                            mode.
                        </p>
                    </div>
                </div>
            </section>

            {/* CALL TO ACTION */}
            <section className="section cta">
                <h2>Start Using Veltaros</h2>
                <p>Create a wallet, explore the network, and begin interacting with the Veltaros transaction flow.</p>

                <Link to="/wallet" className="btn primary large">
                    Get Started
                </Link>
            </section>
        </main>
    );
}
