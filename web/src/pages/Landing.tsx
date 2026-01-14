import React from "react";
import { Link } from "react-router-dom";
import Card from "../components/Card";
import logoUrl from "../assets/veltaros-mark.svg";

export default function Landing(): React.ReactElement {
    return (
        <section className="page">
            <div className="hero">
                <div className="heroLeft">
                    <div className="heroBadge">
                        <span className="pill">Decentralized Currency</span>
                        <span className="pill mutedPill">Go Core • React UI</span>
                    </div>

                    <h1 className="heroTitle">
                        Veltaros
                        <span className="heroTitleSub"> — a production-minded blockchain ecosystem</span>
                    </h1>

                    <p className="heroSubtitle">
                        Full-stack implementation: Go-based node (P2P, consensus, ledger) and a modern web interface for wallet and
                        network visibility.
                    </p>

                    <div className="heroActions">
                        <Link className="btn primary" to="/wallet">
                            Open Wallet
                        </Link>
                        <a className="btn" href="https://github.com/VeltarosLabs" target="_blank" rel="noreferrer noopener">
                            View on GitHub
                        </a>
                    </div>

                    <div className="heroStats">
                        <div className="stat">
                            <div className="statK">Security-first</div>
                            <div className="statV">Handshake, scoring, rate limits</div>
                        </div>
                        <div className="stat">
                            <div className="statK">Clean architecture</div>
                            <div className="statV">Modular Go + typed TS client</div>
                        </div>
                        <div className="stat">
                            <div className="statK">Deploy-ready</div>
                            <div className="statV">Vercel frontend, local node API</div>
                        </div>
                    </div>
                </div>

                <div className="heroRight" aria-hidden="true">
                    <div className="heroMarkWrap">
                        <img className="heroMark" src={logoUrl} alt="" />
                        <div className="heroMarkGlow" />
                    </div>
                </div>
            </div>

            <div className="sectionHeader">
                <h2>Explore</h2>
                <p className="muted">Pick where you want to go.</p>
            </div>

            <div className="gridCards">
                <Card
                    title="Wallet"
                    subtitle="Create / unlock a local encrypted wallet vault, draft & sign transactions."
                    actions={<Link className="btn small primary" to="/wallet">Open</Link>}
                >
                    <ul className="list">
                        <li>Local encryption (PBKDF2 + AES-GCM)</li>
                        <li>Address checksum validation</li>
                        <li>Signed tx draft + node validate/broadcast</li>
                    </ul>
                </Card>

                <Card
                    title="Node & Network"
                    subtitle="View node status, peers, and mempool in real-time."
                    actions={<Link className="btn small" to="/wallet">View</Link>}
                >
                    <ul className="list">
                        <li>P2P identity + challenge-response handshake</li>
                        <li>Peer discovery, backoff/jitter, banlist</li>
                        <li>HTTP status endpoints for operations</li>
                    </ul>
                </Card>

                <Card
                    title="Developer"
                    subtitle="Go module, clean repo structure, MIT license."
                    actions={
                        <a className="btn small" href="https://github.com/VeltarosLabs/Veltaros" target="_blank" rel="noreferrer noopener">
                            Repo
                        </a>
                    }
                >
                    <ul className="list">
                        <li>Go core: internal/ + cmd/ layout</li>
                        <li>Typed API client patterns</li>
                        <li>Production-minded defaults</li>
                    </ul>
                </Card>
            </div>
        </section>
    );
}
