# Veltaros

Veltaros is a decentralized digital currency and full-stack blockchain ecosystem.

This repository contains:
- A **Go core node** (P2P networking, identity handshake, peer discovery, transaction validation, mempool)
- A **React web interface** (landing page + wallet UI with local encrypted vault, transaction signing, network views)

Veltaros is built with a production mindset: clean boundaries, strict validation, security-oriented defaults, and a structure designed to scale.

---

## Repository Structure

- `cmd/`
  - `veltaros-node/` — Node entry point (HTTP API, P2P, mempool, validation)
  - `veltaros-cli/` — CLI tooling (wallet utilities, signing helpers)
- `internal/`
  - `blockchain/` — Blocks, transactions, address rules, mempool + nonce policy (early-stage)
  - `p2p/` — Peer lifecycle, handshake, scoring, peer discovery, banlist, persistence
  - `config/` — Node configuration + flags
  - `logging/` — Structured logging
  - `storage/` — Storage helpers (expanded in later phases)
  - `api/` — API utilities (rate limiting)
- `pkg/`
  - `api/` — Go HTTP API client for node endpoints
  - `types/` — Shared public types (reserved)
  - `version/` — Build version info
- `web/`
  - React frontend (landing + wallet UI)

---

## Current Features

### Node (Go)
- P2P listener with:
  - identity-based HELLO handshake
  - challenge-response verification
  - peer discovery (GET_PEERS / PEERS)
  - scoring + banlist persistence
  - dial backoff + jitter
- HTTP API:
  - `/healthz`, `/version`, `/status`, `/peers`
  - `/mempool`
  - `/tx/validate`
  - `/tx/broadcast`
- Transaction rules (current phase):
  - signed tx draft format (Ed25519)
  - strict address checksum validation
  - signer public key must match `draft.from`
  - per-address nonce tracking to prevent replay

### Web (React)
- Landing page:
  - light/dark mode toggle
  - responsive hero wallpaper + clear sections
- Wallet UI:
  - encrypted local wallet vault (PBKDF2 + AES-GCM)
  - receive address with copy
  - transaction drafting + signing modal
  - local transaction history
  - node status + peers + mempool views
  - responsive layout for mobile/tablet/desktop

---

## Getting Started

### Requirements
- Go (installed)
- Node.js + npm (installed)
- Git

### Run the Node
From the repository root:

```bash
go mod tidy
go run ./cmd/veltaros-node --p2p.network veltaros-testnet --api.listen 127.0.0.1:8080
