package blockchain

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	vcrypto "github.com/VeltarosLabs/Veltaros/internal/crypto"
)

const (
	TxVersion uint32 = 1
)

// TxDraft is the unsigned transaction intent. It is what gets hashed/signature-bound.
type TxDraft struct {
	Version   uint32 `json:"version"`
	NetworkID string `json:"networkId"`

	From string `json:"from"` // sender address
	To   string `json:"to"`   // recipient address

	Amount uint64 `json:"amount"` // smallest unit
	Fee    uint64 `json:"fee"`    // smallest unit

	Nonce     uint64 `json:"nonce"`     // anti-replay per-account/identity (ledger-defined later)
	Timestamp int64  `json:"timestamp"` // unix sec

	Memo string `json:"memo,omitempty"`
}

// SignedTx carries the draft plus the signer identity.
type SignedTx struct {
	Draft       TxDraft `json:"draft"`
	PublicKeyHex string `json:"publicKeyHex"` // ed25519 public key hex (32 bytes)
	SignatureHex string `json:"signatureHex"` // ed25519 signature hex (64 bytes)
	TxID         string `json:"txId"`          // hex of tx hash (double-sha256 of canonical draft bytes)
}

// CanonicalDraftBytes produces stable bytes for hashing/signing.
// We keep a strict, minimal canonical JSON encoding: no whitespace, sorted keys via struct marshaling.
func CanonicalDraftBytes(d TxDraft) ([]byte, error) {
	// Enforce version at encoding time
	if d.Version == 0 {
		d.Version = TxVersion
	}

	// Marshal with stdlib: struct field order is stable; output is deterministic for same values.
	// Important: do not use map encoding here.
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// TxHash = doubleSha256(canonicalDraftBytes)
func TxHash(d TxDraft) ([32]byte, error) {
	b, err := CanonicalDraftBytes(d)
	if err != nil {
		return [32]byte{}, err
	}
	return vcrypto.DoubleSha256(b), nil
}

// SignatureMessage = sha256("veltaros-tx-sign" || networkID || txHash)
// Domain separated and network-bound.
func SignatureMessage(networkID string, txHash [32]byte) [32]byte {
	domain := []byte("veltaros-tx-sign")
	msg := make([]byte, 0, len(domain)+len(networkID)+32)
	msg = append(msg, domain...)
	msg = append(msg, []byte(networkID)...)
	msg = append(msg, txHash[:]...)
	return vcrypto.Sha256(msg)
}

func SignDraft(priv ed25519.PrivateKey, d TxDraft) (SignedTx, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return SignedTx{}, errors.New("invalid ed25519 private key size")
	}
	if d.Timestamp == 0 {
		d.Timestamp = time.Now().UTC().Unix()
	}
	if d.Version == 0 {
		d.Version = TxVersion
	}
	h, err := TxHash(d)
	if err != nil {
		return SignedTx{}, err
	}
	sm := SignatureMessage(d.NetworkID, h)
	sig := ed25519.Sign(priv, sm[:])

	pub := priv.Public().(ed25519.PublicKey)
	return SignedTx{
		Draft:        d,
		PublicKeyHex: hex.EncodeToString(pub),
		SignatureHex: hex.EncodeToString(sig),
		TxID:         hex.EncodeToString(h[:]),
	}, nil
}

func ValidateSignedTx(st SignedTx) error {
	if st.Draft.Version != TxVersion {
		return fmt.Errorf("unsupported tx version: %d", st.Draft.Version)
	}
	if st.Draft.NetworkID == "" {
		return errors.New("networkId is required")
	}
	if st.Draft.From == "" || st.Draft.To == "" {
		return errors.New("from/to required")
	}
	if st.Draft.Amount == 0 {
		return errors.New("amount must be > 0")
	}
	if st.Draft.Fee > st.Draft.Amount {
		return errors.New("fee must be <= amount")
	}
	if st.Draft.Timestamp <= 0 {
		return errors.New("timestamp required")
	}
	// Basic memo length bound (DoS protection)
	if len(st.Draft.Memo) > 256 {
		return errors.New("memo too long")
	}

	pubBytes, err := hex.DecodeString(st.PublicKeyHex)
	if err != nil {
		return errors.New("invalid publicKeyHex")
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return errors.New("invalid publicKeyHex size")
	}
	sigBytes, err := hex.DecodeString(st.SignatureHex)
	if err != nil {
		return errors.New("invalid signatureHex")
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return errors.New("invalid signatureHex size")
	}

	h, err := TxHash(st.Draft)
	if err != nil {
		return err
	}
	if hex.EncodeToString(h[:]) != st.TxID {
		return errors.New("txId mismatch")
	}

	sm := SignatureMessage(st.Draft.NetworkID, h)
	if !ed25519.Verify(ed25519.PublicKey(pubBytes), sm[:], sigBytes) {
		return errors.New("invalid signature")
	}

	// NOTE: Ledger checks (balance/nonce, etc.) come later.
	return nil
}
