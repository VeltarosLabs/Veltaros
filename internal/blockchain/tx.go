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

	// Policy bounds (node-level, can be made configurable later)
	MaxMemoLen       = 256
	MaxFutureSkewSec = 5 * 60    // 5 minutes
	MaxPastSkewSec   = 24 * 3600 // 24 hours
	MinFee           = 1
	MaxTxAmount      = ^uint64(0)
)

type TxDraft struct {
	Version   uint32 `json:"version"`
	NetworkID string `json:"networkId"`

	From string `json:"from"`
	To   string `json:"to"`

	Amount uint64 `json:"amount"`
	Fee    uint64 `json:"fee"`

	Nonce     uint64 `json:"nonce"`
	Timestamp int64  `json:"timestamp"`

	Memo string `json:"memo,omitempty"`
}

type SignedTx struct {
	Draft        TxDraft `json:"draft"`
	PublicKeyHex string  `json:"publicKeyHex"` // ed25519 raw pubkey hex (32 bytes)
	SignatureHex string  `json:"signatureHex"` // ed25519 signature hex (64 bytes)
	TxID         string  `json:"txId"`         // hex doubleSha256(canonicalDraftBytes)
}

func CanonicalDraftBytes(d TxDraft) ([]byte, error) {
	if d.Version == 0 {
		d.Version = TxVersion
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func TxHash(d TxDraft) ([32]byte, error) {
	b, err := CanonicalDraftBytes(d)
	if err != nil {
		return [32]byte{}, err
	}
	return vcrypto.DoubleSha256(b), nil
}

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
	d := st.Draft

	if d.Version != TxVersion {
		return fmt.Errorf("unsupported tx version: %d", d.Version)
	}
	if d.NetworkID == "" {
		return errors.New("networkId is required")
	}

	// Address format validation
	if err := ValidateAddress(d.From); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	if err := ValidateAddress(d.To); err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}
	if d.From == d.To {
		return errors.New("from and to must differ")
	}

	if d.Amount == 0 || d.Amount > MaxTxAmount {
		return errors.New("amount must be > 0")
	}
	if d.Fee < MinFee {
		return fmt.Errorf("fee must be >= %d", MinFee)
	}
	if d.Fee > d.Amount {
		return errors.New("fee must be <= amount")
	}
	if d.Nonce == 0 {
		return errors.New("nonce must be > 0")
	}
	if d.Timestamp <= 0 {
		return errors.New("timestamp required")
	}

	if len(d.Memo) > MaxMemoLen {
		return errors.New("memo too long")
	}

	// Timestamp skew policy
	now := time.Now().UTC().Unix()
	if d.Timestamp > now+MaxFutureSkewSec {
		return errors.New("timestamp too far in future")
	}
	if d.Timestamp < now-MaxPastSkewSec {
		return errors.New("timestamp too far in past")
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

	h, err := TxHash(d)
	if err != nil {
		return err
	}
	if hex.EncodeToString(h[:]) != st.TxID {
		return errors.New("txId mismatch")
	}

	sm := SignatureMessage(d.NetworkID, h)
	if !ed25519.Verify(ed25519.PublicKey(pubBytes), sm[:], sigBytes) {
		return errors.New("invalid signature")
	}

	return nil
}
