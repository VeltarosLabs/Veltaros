# Veltaros

Veltaros is a decentralized digital currency and full-stack blockchain ecosystem.

This repository contains:
- A Go-based node (P2P networking, identity handshake, peer discovery, transaction validation, mempool, block production in dev mode)
- A React web interface (landing page, wallet UI, explorer)

Veltaros is built with a production mindset: clear architecture, strict validation, security-oriented defaults, and a structure designed to evolve into a complete blockchain network.

---

## Repository Structure

- `cmd/`
  - `veltaros-node/` — Node entry point (HTTP API + P2P)
  - `veltaros-cli/` — CLI tooling
- `internal/`
  - `blockchain/` — blocks, merkle root, tx validation, nonce policy, block store (early stage)
  - `ledger/` — balances + staged mempool spending (early stage)
  - `p2p/` — peer handshake, discovery, scoring, banlist, persistence
  - `config/` — flags + environment configuration
  - `api/` — server utilities (rate limiting, security middleware)
  - `logging/`, `storage/`, `crypto/`
- `pkg/`
  - `api/` — Go client for node HTTP endpoints
  - `version/`
- `web/`
  - React frontend (Landing + Wallet + Explorer)

---

## Current Features

### Node (Go)
- P2P:
  - identity HELLO + challenge-response verification
  - peer discovery + dial backoff + banlist + peer store
  - scoring + persistence
- HTTP API:
  - `/healthz`, `/version`, `/status`, `/peers`
  - `/mempool`, `/account/<address>`
  - `/tx/validate`, `/tx/broadcast`
  - `/tip`, `/blocks`, `/block/<hash>` (explorer endpoints)
- Security:
  - CORS allowlist
  - optional API key for tx endpoints
  - rate limiting on transaction routes
- Ledger (early stage):
  - confirmed balances persisted to disk
  - spendable balance uses staged mempool spending
- Dev mode:
  - `/dev/produce-block` confirms mempool txs into blocks (dev only)

### Web (React)
- Dark/Light theme toggle
- Landing page with hero wallpaper and clean cards
- Wallet:
  - local encrypted vault (PBKDF2 + AES-GCM)
  - receive address + copy
  - send flow with signing modal
  - nonce + balance display (from node account endpoint)
  - optional faucet support (appears only if node enables it)
- Explorer:
  - tip + recent blocks
  - block details modal
  - transaction lookup inside recent blocks

---

## Run Locally

### Node
From repository root:

```bash
go mod tidy
go run ./cmd/veltaros-node --p2p.network veltaros-testnet --api.listen 127.0.0.1:8080
