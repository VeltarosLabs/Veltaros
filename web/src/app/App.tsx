import React from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import Landing from "../pages/Landing";
import Wallet from "../pages/Wallet";
import Navbar from "../components/Navbar";

export default function App(): React.ReactElement {
    return (
        <div className="app">
            <Navbar />
            <main className="container">
                <Routes>
                    <Route path="/" element={<Landing />} />
                    <Route path="/wallet" element={<Wallet />} />
                    <Route path="*" element={<Navigate to="/" replace />} />
                </Routes>
            </main>
            <footer className="footer">
                <div className="footer-inner">
                    <span>© {new Date().getFullYear()} VeltarosLabs</span>
                </div>
            </footer>
        </div>
    );
}
