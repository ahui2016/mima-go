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
)

func newNonce() (nonce Nonce, err error) {
	_, err = rand.Read(nonce[:])
	return
}

func randomString() string {
	someBytes := make([]byte, 255)
	if _, err := rand.Read(someBytes); err != nil {
		panic(err) // 因为这里有错误的可能性极小, 因此偷懒不处理.
	}
	return base64.StdEncoding.EncodeToString(someBytes)
}

func dbFileIsNotExist() bool {
	if _, err := os.Stat(dbFullPath); os.IsNotExist(err) {
		return true
	}
	return false
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
func writeFragFile(box64 string) error {
	fragmentPath := filepath.Join(dbDirPath, newFragmentName())
	return writeFile(fragmentPath, box64)
}

// 把数据写到指定位置.
func writeFile(fullpath string, box64 string) error {
	return ioutil.WriteFile(fullpath, []byte(box64), 0644)
}

// backupToTar 把数据库文件以及碎片文件备份到一个 tarball 里.
// 主要在 Rebuild 之前使用, 以防万一 rebuild 出错.
// 为了方便测试返回 tarball 的完整路径.
func backupToTar(files []string) (filePath string, err error) {
	filePath = filepath.Join(dbDirPath, newBackupName())
	err = tarball.Create(filePath, files)
	return
}

// filesToBackup 返回需要备份的文件的完整路径.
func filesToBackup(fragFiles []string) []string {
	return append(fragFiles, dbFullPath)
}

// fragFilePaths 返回数据库碎片文件的完整路径, 并且已排序.
func fragFilePaths() ([]string, error) {
	pattern := filepath.Join(dbDirPath, "*"+FragExt)
	filePaths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(filePaths)
	return filePaths, nil
}

func readAndDecrypt(fullpath string, key SecretKey) (*Mima, error) {
	b, err := ioutil.ReadFile(fullpath)
	if err != nil {
		return nil, err
	}
	box64 := string(b)
	return DecryptToMima(box64, key)
}

// writeln 主要用于把已加密的 box 逐行写入文件 (添加换行符).
// 这里需要把二进制数据转换为 base64, 因为二进制数据会与换行符冲突.
func bufWriteln(w *bufio.Writer, box64 string) error {
	if _, err := w.WriteString(box64 + "\n"); err != nil {
		return err
	}
	return nil
}
