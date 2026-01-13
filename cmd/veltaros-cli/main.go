package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	vcrypto "github.com/VeltarosLabs/Veltaros/internal/crypto"
	"github.com/VeltarosLabs/Veltaros/internal/wallet"
	"github.com/VeltarosLabs/Veltaros/pkg/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "version":
		runVersion()
	case "wallet":
		runWallet(os.Args[2:])
	case "sign":
		runSign(os.Args[2:])
	case "verify":
		runVerify(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Print(`Veltaros CLI

Usage:
  veltaros-cli version
  veltaros-cli wallet new --out <path>
  veltaros-cli wallet address --key <path>
  veltaros-cli sign --key <path> --msg <text>
  veltaros-cli verify --pub <hex> --msg <text> --sig <hex>

Notes:
  - wallet keys are stored as hex-encoded ed25519 private keys (64 bytes).
  - addresses are deterministic: hex(pubHash20||checksum4).
`)
}

func runVersion() {
	v := version.Get()
	fmt.Printf("Veltaros CLI\nVersion: %s\nCommit:  %s\nGo:      %s\nTarget:  %s\n",
		v.Version, v.Commit, v.GoVersion, v.Platform)
}

func runWallet(args []string) {
	if len(args) < 1 {
		usage()
		os.Exit(2)
	}

	switch args[0] {
	case "new":
		fs := flag.NewFlagSet("wallet new", flag.ExitOnError)
		out := fs.String("out", filepath.Join("data", "wallets", "default.key"), "Output path for private key file")
		_ = fs.Parse(args[1:])

		kp, err := wallet.Generate()
		if err != nil {
			fatal(err)
		}
		if err := wallet.SavePrivateKeyHex(*out, kp.PrivateKey); err != nil {
			fatal(err)
		}
		addr, err := wallet.AddressFromPublicKey(kp.PublicKey)
		if err != nil {
			fatal(err)
		}
		fmt.Println("Saved private key:", *out)
		fmt.Println("Address:", addr)

	case "address":
		fs := flag.NewFlagSet("wallet address", flag.ExitOnError)
		keyPath := fs.String("key", filepath.Join("data", "wallets", "default.key"), "Path to private key file")
		_ = fs.Parse(args[1:])

		priv, err := wallet.LoadPrivateKeyHex(*keyPath)
		if err != nil {
			fatal(err)
		}
		pubAny := priv.Public()
		pub, ok := pubAny.([]byte)
		if ok {
			addr, err := wallet.AddressFromPublicKey(pub)
			if err != nil {
				fatal(err)
			}
			fmt.Println(addr)
			return
		}

		// ed25519.PrivateKey.Public() returns ed25519.PublicKey (alias []byte)
		addr, err := wallet.AddressFromPublicKey(pubAny.([]byte))
		if err != nil {
			fatal(err)
		}
		fmt.Println(addr)

	default:
		usage()
		os.Exit(2)
	}
}

func runSign(args []string) {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	keyPath := fs.String("key", filepath.Join("data", "wallets", "default.key"), "Path to private key file")
	msg := fs.String("msg", "", "Message to sign")
	_ = fs.Parse(args)

	if strings.TrimSpace(*msg) == "" {
		fatal(fmt.Errorf("--msg is required"))
	}

	priv, err := wallet.LoadPrivateKeyHex(*keyPath)
	if err != nil {
		fatal(err)
	}

	sig, err := vcrypto.SignEd25519(priv, []byte(*msg))
	if err != nil {
		fatal(err)
	}

	fmt.Println(hex.EncodeToString(sig))
}

func runVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	pubHex := fs.String("pub", "", "Public key hex (32 bytes)")
	msg := fs.String("msg", "", "Message to verify")
	sigHex := fs.String("sig", "", "Signature hex (64 bytes)")
	_ = fs.Parse(args)

	if strings.TrimSpace(*pubHex) == "" || strings.TrimSpace(*sigHex) == "" || strings.TrimSpace(*msg) == "" {
		fatal(fmt.Errorf("--pub, --sig, and --msg are required"))
	}

	pub, err := vcrypto.DecodeHex(strings.TrimSpace(*pubHex))
	if err != nil {
		fatal(err)
	}
	sig, err := vcrypto.DecodeHex(strings.TrimSpace(*sigHex))
	if err != nil {
		fatal(err)
	}

	ok := vcrypto.VerifyEd25519(pub, []byte(*msg), sig)
	if ok {
		fmt.Println("OK")
		return
	}
	fmt.Println("INVALID")
	os.Exit(1)
}

func fatal(err error) {
	_, _ = os.Stderr.WriteString("veltaros-cli error: " + err.Error() + "\n")
	os.Exit(1)
}
