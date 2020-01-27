package db

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"time"
)

const (
	KeySize   = 32
	NonceSize = 24

	DateTimeFormat = "2006-01-02 15:04:05"

	// 数据库碎片文件的后缀名
	FragExt = ".db.frag"

	// 数据库备份文件的后缀名
	TarballExt = ".tar.gz"
)

var (
	FileNotFound      error
	errNeedTitle      error
	ErrAliasConflicts error
)

type (
	Nonce     = [NonceSize]byte
	SecretKey = [KeySize]byte
)

func init() {
	FileNotFound = errors.New("找不到数据库文件")
	errNeedTitle = errors.New("'Title' 长度不可为零, 请填写 Title")
	ErrAliasConflicts = errors.New("该 alias 已存在")
}

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

func writeFile(fullPath string, box64 string) error {
	return ioutil.WriteFile(fullPath, []byte(box64), 0644)
}

func newNonce() (nonce Nonce, err error) {
	_, err = rand.Read(nonce[:])
	return
}

// newID 返回一个由时间戳和随机数组成的 id, 经测试瞬间生成一万个 id 不会重复.
// 由于时间戳的精度为秒, 因此如果两次生成 id 之间超过一秒, 则绝对不会重复.
func newID() (id string, err error) {
	var max int64 = 100_000_000
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return
	}
	timestamp := time.Now().Unix()
	idInt64 := timestamp*max + n.Int64()
	id = strconv.FormatInt(idInt64, 36)
	return
}

func bytesToKey(b []byte) (key SecretKey) {
	copy(key[:], b)
	return
}

func newTimestampFilename(ext string) string {
	name := strconv.FormatInt(time.Now().UnixNano(), 10)
	return name + ext
}

func readAndDecrypt(fullPath string, key *SecretKey) (mima *Mima, err error) {
	var b []byte
	if b, err = ioutil.ReadFile(fullPath); err != nil {
		return
	}
	box64 := string(b)
	return Decrypt(box64, key)
}

// bufWriteln 主要用于把已加密的 box 逐行写入文件 (添加换行符).
// 因为二进制数据会与换行符冲突, 因此这里需要把二进制数据转换为 base64.
func bufWriteln(w *bufio.Writer, box64 string) error {
	if _, err := w.WriteString(box64 + "\n"); err != nil {
		return err
	}
	return nil
}

func deleteFiles(filePaths []string) error {
	for _, f := range filePaths {
		if err := os.Remove(f); err != nil {
			return err
		}
	}
	return nil
}
