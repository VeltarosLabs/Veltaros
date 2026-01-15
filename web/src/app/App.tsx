import React from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import Navbar from "../components/Navbar";
import Landing from "../pages/Landing";
import Wallet from "../pages/Wallet";
import SocialLinks from "../components/SocialLinks";
import { useTheme } from "../hooks/useTheme";
import logoPng from "../assets/logo.png";

export default function App(): React.ReactElement {
    const { theme, toggle } = useTheme();

    return (
        <div className="appShell">
            <a href="#main-content" className="skipLink">
                Skip to content
            </a>

            <Navbar theme={theme} onToggleTheme={toggle} />

            <main id="main-content" className="container main" role="main" tabIndex={-1}>
                <Routes>
                    <Route path="/" element={<Landing />} />
                    <Route path="/wallet" element={<Wallet />} />
                    <Route path="*" element={<Navigate to="/" replace />} />
                </Routes>
            </main>

            <footer className="siteFooter">
                <div className="container footerWrap">
                    <div className="footerLeft">
                        <div className="footerBrand">
                            <img src={logoPng} alt="Veltaros logo" className="footerLogo" />
                            <div className="footerBrandText">
                                <div className="footerTitle">Veltaros</div>
                                <div className="footerSubtitle">Decentralized digital currency ecosystem</div>
                            </div>
                        </div>

                        <div className="footerMeta">
                            <span>© {new Date().getFullYear()} Veltaros</span>
                            <span className="dot">•</span>
                            <span>All rights reserved</span>
                        </div>
                    </div>

                    <div className="footerRight">
                        <div className="footerSectionTitle">Community</div>
                        <SocialLinks />
                    </div>
                </div>
            </footer>
        </div>
    );
}
