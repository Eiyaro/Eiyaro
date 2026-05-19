package main

import (
	"fmt"

	"github.com/Eiyaro/Eiyaro/util"
	"github.com/kaspanet/go-secp256k1"
)

func main() {
	cfg, err := parseConfig()
	if err != nil {
		panic(err)
	}

	keyPair, err := secp256k1.GenerateSchnorrKeyPair()
	if err != nil {
		panic(err)
	}

	publicKey, err := keyPair.SchnorrPublicKey()
	if err != nil {
		panic(err)
	}

	pubKeySerialized, err := publicKey.Serialize()
	if err != nil {
		panic(err)
	}

	addr, err := util.NewAddressPublicKey(pubKeySerialized[:], cfg.NetParams().Prefix)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Private key: %x\n", keyPair.SerializePrivateKey())
	fmt.Printf("Address: %s\n", addr)
}
