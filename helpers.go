package main

import (
	"crypto/rand"
)

func newNonce() (nonce [NonceSize]byte) {
	if _, err := rand.Read(nonce[:]); err != nil {
		panic(err)
	}
	return
}
