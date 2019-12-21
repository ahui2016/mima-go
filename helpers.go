package main

import (
	"crypto/rand"
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
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

// 确保数据库文件不存在, 如果已存在则报错.
func dbMustNotExist() {
	if _, err := os.Stat(dbFullPath); !os.IsNotExist(err) {
		panic("数据库文件已存在, 不可重复创建")
	}
}

// 确保数据库文件已存在, 如果不存在则报错.
func dbFileMustExist() {
	if _, err := os.Stat(dbFullPath); os.IsNotExist(err) {
		panic("需要数据库文件, 但不存在")
	}
}

// newNameByNow 返回当前时间戳的字符串.
func newNameByNow() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// newFragmentName 返回一个新的数据库碎片文件名.
func newFragmentName() string {
	return newNameByNow() + FragExt
}

// newBackupName 返回一个新的备份文件名.
func newBackupName() string {
	return newNameByNow() + tarballExt
}

// 把已加密的数据写到一个新文件中 (即生成一个新的数据库碎片).
func writeFragFile(sealed []byte) {
	fragmentPath := filepath.Join(dbDirPath, newFragmentName())
	writeFile(fragmentPath, sealed)
}

// 把数据写到指定位置.
func writeFile(fullpath string, sealed []byte) {
	if err := ioutil.WriteFile(fullpath, sealed, 0644); err != nil {
		panic(err)
	}
}
