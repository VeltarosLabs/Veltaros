import React from "react";
import { Link } from "react-router-dom";
import logoPng from "../assets/logo.png";
import heroImg from "../assets/hero.jpg";
import SocialLinks from "../components/SocialLinks";
import "../styles/landing.css";

export default function Landing(): React.ReactElement {
    return (
        <main className="landing">
            {/* HERO */}
            <section
                className="hero"
                style={{ backgroundImage: `url(${heroImg})` }}
            >
                <div className="heroOverlay" />

                <div className="heroContent">
                    <img
                        src={logoPng}
                        alt="Veltaros logo"
                        className="heroLogo"
                    />

                    <h1 className="heroTitle">Veltaros</h1>
                    <p className="heroSubtitle">
                        A decentralized digital currency focused on security, transparency,
                        and long-term sustainability.
                    </p>

                    <div className="heroActions">
                        <Link to="/wallet" className="btn primary">
                            Open Wallet
                        </Link>
                        <a
                            href="#vision"
                            className="btn ghost"
                        >
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
                        Veltaros is built as a complete blockchain ecosystem — not an
                        experiment, not a demo.
                    </p>
                </header>

                <div className="cardGrid">
                    <div className="card">
                        <h3>Decentralized by Design</h3>
                        <p>
                            A peer-to-peer network with independent nodes, transparent
                            consensus, and no central authority.
                        </p>
                    </div>

                    <div className="card">
                        <h3>Security First</h3>
                        <p>
                            Modern cryptography, signed transactions, replay protection,
                            and strict validation rules at the protocol level.
                        </p>
                    </div>

                    <div className="card">
                        <h3>Built for the Long Term</h3>
                        <p>
                            Clean architecture, audited logic, and a focus on maintainability
                            rather than hype.
                        </p>
                    </div>
                </div>
            </section>

            {/* TECHNOLOGY */}
            <section className="section alt">
                <header className="sectionHeader">
                    <h2>Technology</h2>
                    <p>
                        A full-stack blockchain system designed for real-world use.
                    </p>
                </header>

                <div className="cardGrid">
                    <div className="card">
                        <h3>Core Engine</h3>
                        <p>
                            Written in Go for performance, concurrency, and reliability.
                            Includes networking, consensus, and ledger logic.
                        </p>
                    </div>

                    <div className="card">
                        <h3>Wallet & Interface</h3>
                        <p>
                            A modern web wallet built with React, designed for clarity,
                            safety, and ease of use across all devices.
                        </p>
                    </div>

                    <div className="card">
                        <h3>Open Ecosystem</h3>
                        <p>
                            Modular architecture that allows future tools, explorers,
                            and integrations to grow naturally.
                        </p>
                    </div>
                </div>
            </section>

            {/* CALL TO ACTION */}
            <section className="section cta">
                <h2>Start Using Veltaros</h2>
                <p>
                    Create a wallet, explore the network, and participate in a
                    decentralized financial system built with intention.
                </p>

                <Link to="/wallet" className="btn primary large">
                    Get Started
                </Link>
            </section>

            {/* FOOTER */}
            <footer className="footer">
                <div className="footerInner">
                    <div className="footerBrand">
                        <img
                            src={logoPng}
                            alt="Veltaros logo"
                            className="footerLogo"
                        />
                        <span className="footerName">Veltaros</span>
                    </div>

                    <SocialLinks />

                    <p className="footerNote">
                        © {new Date().getFullYear()} Veltaros. All rights reserved.
                    </p>
                </div>
            </footer>
        </main>
    );
}
