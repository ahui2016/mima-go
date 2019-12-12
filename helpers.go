package main

import (
	"crypto/rand"
)

func newNonce() (nonce Nonce) {
	if _, err := rand.Read(nonce[:]); err != nil {
		panic(err)
	}
	return
}
