package db

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io/ioutil"
)

const (
	KeySize      = 32
	NonceSize    = 24

	// 数据库碎片文件的后缀名
	FragExt = ".db.frag"

	// 数据库备份文件的后缀名
	TarballExt = ".tar.gz"
)

// Nonce 是 [NonceSize]byte 的别名.
type Nonce = [NonceSize]byte

// SecretKey 是 [KeySize]byte 的别名.
type SecretKey = [KeySize]byte


func newRandomKey() SecretKey {
	s := randomString()
	key := sha256.Sum256([]byte(s))
	return key
}

func randomString() string {
	someBytes := make([]byte, 255)
	if _, err := rand.Read(someBytes); err != nil {
		panic(err) // 因为这里有错误的可能性极小, 因此偷懒不处理.
	}
	return base64.StdEncoding.EncodeToString(someBytes)
}

func writeFile(fullPath string, box64bytes []byte) error {
	return ioutil.WriteFile(fullPath, box64bytes, 0644)
}
