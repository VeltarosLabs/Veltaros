import React from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import Navbar from "../components/Navbar";
import Landing from "../pages/Landing";
import Wallet from "../pages/Wallet";
import SocialLinks from "../components/SocialLinks";
import { useTheme } from "../hooks/useTheme";

export default function App(): React.ReactElement {
    const { theme, toggle } = useTheme();

    return (
        <div className="app">
            <Navbar theme={theme} onToggleTheme={toggle} />

            <main className="container main">
                <Routes>
                    <Route path="/" element={<Landing />} />
                    <Route path="/wallet" element={<Wallet />} />
                    <Route path="*" element={<Navigate to="/" replace />} />
                </Routes>
            </main>

            <footer className="footer">
                <div className="footerInner">
                    <div>
                        <div className="footerTitle">Veltaros</div>
                        <div className="footerSub">Decentralized currency ecosystem</div>
                    </div>

                    <SocialLinks />
                </div>
            </footer>
        </div>
    );
}
