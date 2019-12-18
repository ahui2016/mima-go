package main

import (
	"crypto/rand"
	"encoding/base64"
	"strconv"
	"time"
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

// NewFragmentName 返回一个新的数据库碎片文件名.
func newFragmentName() string {
	name := strconv.FormatInt(time.Now().UnixNano(), 10)
	return name + FragExt
}
