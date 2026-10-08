package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
)

func main() {
	path := "deploy/local/.env"
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintln(os.Stderr, "local .env already exists; refusing to overwrite it")
		os.Exit(1)
	} else if !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	seed := make([]byte, ed25519.SeedSize)
	secret := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		panic(err)
	}
	if _, err := rand.Read(secret); err != nil {
		panic(err)
	}
	key := ed25519.NewKeyFromSeed(seed)
	content := fmt.Sprintf("# Local development only. Never commit this file.\nCORE_SIGNING_SEED_B64=%s\nAPI_VERIFY_PUBLIC_KEY_B64=%s\nINTERNAL_SERVICE_SECRET_B64=%s\nFRONTEND_ORIGIN=http://localhost:5173\nSMTP_FROM=atethat@localhost\n",
		base64.StdEncoding.EncodeToString(seed), base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey)), base64.StdEncoding.EncodeToString(secret))
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		panic(err)
	}
	fmt.Println("Created", path, "with local-only random credentials")
}
