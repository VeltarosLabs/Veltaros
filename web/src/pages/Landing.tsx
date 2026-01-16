import React from "react";
import { Link } from "react-router-dom";
import logoPng from "../assets/logo.png";
import heroImg from "../assets/hero.jpg";
import "../styles/landing.css";
import TipPreview from "../components/TipPreview";

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

            {/* NETWORK */}
            <section className="section alt">
                <header className="sectionHeader">
                    <h2>Network</h2>
                    <p>
                        Observe the chain tip and explore recent blocks directly from your node.
                    </p>
                </header>

                <div className="cardGrid">
                    <div className="card">
                        <h3>Explorer</h3>
                        <p>
                            Review chain progress and open blocks to inspect headers and transactions.
                        </p>

                        <div className="tipWrap">
                            <TipPreview />
                        </div>

                        <div className="landingActions">
                            <Link to="/explorer" className="btn small primary">
                                Open Explorer
                            </Link>
                            <Link to="/wallet" className="btn small">
                                Wallet
                            </Link>
                        </div>
                    </div>

                    <div className="card">
                        <h3>Transactions</h3>
                        <p>
                            Draft, sign, and broadcast transactions with nonce protection and balance checks.
                        </p>

                        <div className="landingActions">
                            <Link to="/wallet" className="btn small primary">
                                Send
                            </Link>
                            <Link to="/wallet" className="btn small">
                                Receive
                            </Link>
                        </div>
                    </div>

                    <div className="card">
                        <h3>Node Status</h3>
                        <p>
                            View peer connectivity, mempool activity, and operational status in one place.
                        </p>

                        <div className="landingActions">
                            <Link to="/wallet" className="btn small primary">
                                Open Network View
                            </Link>
                        </div>
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
