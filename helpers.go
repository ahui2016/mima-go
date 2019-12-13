package main

import (
	"crypto/rand"
	"encoding/base64"
)

func newNonce() (nonce Nonce) {
	if _, err := rand.Read(nonce[:]); err != nil {
		panic(err)
	}
	return
}

func randomString() string {
	someBytes := make([]byte, 255)
	if _, err := rand.Read(someBytes); err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(someBytes)
}
