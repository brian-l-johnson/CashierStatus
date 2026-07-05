// Command gen-control-key generates the Ed25519 keypair used to sign
// control QR codes. Run once; the seed goes in the server env, the public
// key is committed to merch-app.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
)

func main() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("failed to generate key: %v", err)
	}

	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		log.Fatalf("failed to marshal public key: %v", err)
	}

	fmt.Println("# Add to the CashierStatus server environment (.env / container env):")
	fmt.Printf("CONTROL_SIGNING_KEY=%s\n\n", base64.StdEncoding.EncodeToString(priv.Seed()))
	fmt.Println("# Commit to merch-app as pkg/controlsign/pubkey.pem and keys/control-signing.pubkey:")
	fmt.Print(string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})))
}
