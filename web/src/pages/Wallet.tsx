import React from "react";

export default function Wallet(): React.ReactElement {
    return (
        <section className="page">
            <h2>Wallet UI</h2>
            <p>
                This is the initial wallet interface shell. Next, we’ll wire it to the Veltaros node API and implement secure key
                management patterns.
            </p>

            <div className="card">
                <h3>Status</h3>
                <p>Not connected</p>
            </div>
        </section>
    );
}
