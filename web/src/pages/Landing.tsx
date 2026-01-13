import React from "react";
import { Link } from "react-router-dom";

export default function Landing(): React.ReactElement {
    return (
        <section className="hero">
            <div className="hero-content">
                <h1>Veltaros</h1>
                <p className="subtitle">
                    A production-ready decentralized digital currency and full-stack blockchain ecosystem built for security,
                    scalability, and long-term maintainability.
                </p>

                <div className="cta-row">
                    <Link className="btn primary" to="/wallet">
                        Open Wallet UI
                    </Link>
                    <a className="btn" href="https://github.com/VeltarosLabs" target="_blank" rel="noreferrer noopener">
                        View on GitHub
                    </a>
                </div>

                <div className="cards">
                    <div className="card">
                        <h3>Go Core Engine</h3>
                        <p>P2P networking, consensus, ledger/state, and cryptography designed with clean boundaries.</p>
                    </div>
                    <div className="card">
                        <h3>Web Frontend</h3>
                        <p>React-based landing and wallet UI, deployable to Vercel at veltaros.org.</p>
                    </div>
                    <div className="card">
                        <h3>MIT Licensed</h3>
                        <p>Open and permissive — built for collaboration and transparent development.</p>
                    </div>
                </div>
            </div>
        </section>
    );
}
