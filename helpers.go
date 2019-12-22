package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/ahui2016/mima-go/tarball"
	"github.com/ahui2016/mima-go/util"
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

// backupToTar 把数据库文件以及碎片文件备份到一个 tarball 里.
// 主要在 Rebuild 之前使用, 以防万一 rebuild 出错.
// 为了方便测试返回 tarball 的完整路径.
func backupToTar() (filePath string) {
	files := filesToBackup()
	filePath = filepath.Join(dbDirPath, newBackupName())
	if err := tarball.Create(filePath, files); err != nil {
		panic(err)
	}
	return
}

// filesToBackup 返回需要备份的文件的完整路径.
func filesToBackup() []string {
	filePaths := fragFilePaths()
	return append(filePaths, dbFullPath)
}

// fragFilePaths 返回数据库碎片文件的完整路径, 并且已排序.
func fragFilePaths() []string {
	pattern := filepath.Join(dbDirPath, "*"+FragExt)
	filePaths, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}
	sort.Strings(filePaths)
	return filePaths
}

func readAndDecrypt(fullpath string, key SecretKey) (*Mima, bool) {
	box := util.ReadFile(fullpath)
	return DecryptToMima(box, key)
}

// writeln 主要用于把已加密的 box 逐行写入文件 (添加换行符).
func bufWriteln(w *bufio.Writer, box []byte) error {
	if _, err := w.Write(box); err != nil {
		return err
	}
	if _, err := w.WriteString("\n"); err != nil {
		return err
	}
	return nil
}
