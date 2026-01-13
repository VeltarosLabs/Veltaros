# Veltaros

Veltaros is a production-grade, decentralized digital currency and blockchain ecosystem built for security, scalability, and long-term maintainability.

This repository contains:
- **Go Core Engine**: P2P networking, consensus (PoW/PoS), ledger/state, cryptography, and wallet primitives.
- **Web Frontend**: Node.js/React application for the landing site and wallet UI, deployable on Vercel at **Veltaros.org**.

---

## Vision

Veltaros is designed as a legitimate decentralized monetary network with:

- **Verifiable integrity** through cryptographic validation and deterministic consensus rules
- **Resilience** through peer-to-peer networking and robust node behavior under adverse conditions
- **Security-first engineering** including strict input validation, safe defaults, and minimal attack surface
- **Developer usability** with clean modular design, typed APIs, and maintainable code structure

---

## Repository Structure

- `cmd/`
  - `veltaros-node/` — Full node entry point (networking, consensus, ledger)
  - `veltaros-cli/` — CLI entry point (operator tools, wallet utilities)
- `internal/` — Core implementation (not intended for external import)
  - `blockchain/` — Blocks, transactions, chain rules
  - `consensus/` — Proof-of-Work / Proof-of-Stake logic and validation rules
  - `crypto/` — Hashing, signatures, key utilities
  - `ledger/` — State management and accounting model
  - `p2p/` — Peer discovery, transport, protocol messaging
  - `wallet/` — Wallet primitives and key handling
  - `config/` — Configuration loading and defaults
  - `logging/` — Structured logging
  - `storage/` — Persistence interfaces and implementations
- `pkg/` — Public libraries (reusable packages)
- `web/` — React frontend (Landing + Wallet UI)

---

## Quick Start

### Prerequisites
- Go (installed)
- Node.js (LTS recommended)
- Git

### Go (Core)
```bash
go mod tidy
go test ./...
go build ./...
