import React from "react";
import Card from "../components/Card";
import logoUrl from "../assets/veltaros-mark.svg";
import { Link } from "react-router-dom";

export default function Landing(): React.ReactElement {
    return (
        <section className="page">
            <div className="hero">
                <div className="heroLeft">
                    <div className="badgeRow">
                        <span className="pill">Veltaros Network</span>
                        <span className="pill subtle">Secure • Fast • Transparent</span>
                    </div>

                    <h1 className="heroTitle">
                        Veltaros
                        <span className="heroTitleSub"> — modern decentralized digital currency</span>
                    </h1>

                    <p className="heroSubtitle">
                        A full-stack blockchain ecosystem: a node that powers the network and a clean wallet experience designed for
                        real-world use.
                    </p>

                    <div className="heroActions">
                        <Link className="btn primary" to="/wallet">
                            Open Wallet
                        </Link>
                        <a className="btn" href="#features">
                            Explore Features
                        </a>
                    </div>

                    <div className="heroStats">
                        <div className="stat">
                            <div className="statK">Network</div>
                            <div className="statV">Peer discovery, handshake verification, scoring</div>
                        </div>
                        <div className="stat">
                            <div className="statK">Wallet</div>
                            <div className="statV">Local encrypted vault, signed transactions</div>
                        </div>
                        <div className="stat">
                            <div className="statK">Security</div>
                            <div className="statV">Strict validation, rate-limiting, resilient defaults</div>
                        </div>
                    </div>
                </div>

                <div className="heroRight" aria-hidden="true">
                    <div className="heroMarkWrap">
                        <img className="heroMark" src={logoUrl} alt="" />
                        <div className="heroGlow" />
                    </div>
                </div>
            </div>

            <div id="features" className="sectionHeader">
                <h2>What you can do</h2>
                <p className="muted">Choose where you want to start.</p>
            </div>

            <div className="cardGrid">
                <Card title="Wallet" subtitle="Create and manage your local wallet, then sign transactions.">
                    <ul className="list">
                        <li>Encrypted vault (password protected)</li>
                        <li>Receive address with copy action</li>
                        <li>Draft, sign, validate, broadcast transactions</li>
                    </ul>
                    <div className="cardActions">
                        <Link className="btn small primary" to="/wallet">
                            Open Wallet
                        </Link>
                    </div>
                </Card>

                <Card title="Network" subtitle="View the node’s view of the network and transaction pool.">
                    <ul className="list">
                        <li>Peer list with verification and scoring</li>
                        <li>Mempool viewer</li>
                        <li>Live node status</li>
                    </ul>
                    <div className="cardActions">
                        <Link className="btn small" to="/wallet">
                            Open Network View
                        </Link>
                    </div>
                </Card>

                <Card title="Transactions" subtitle="A clean workflow from draft to broadcast.">
                    <ul className="list">
                        <li>Checksum address validation</li>
                        <li>Signing confirmation modal</li>
                        <li>Local transaction history</li>
                    </ul>
                    <div className="cardActions">
                        <Link className="btn small" to="/wallet">
                            Go to Transactions
                        </Link>
                    </div>
                </Card>
            </div>
        </section>
    );
}
