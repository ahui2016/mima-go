package main

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// 一些常量
const (
	KeySize   = 32
	NonceSize = 24

	DBDir   = "mimadb"
	DBName  = "mima.db"
	TempDir = "temp_dir_for_test"

	// 数据库碎片文件的后缀名
	FragExt = ".db.frag"
)

var (
	baseDir    string
	dbDirPath  string
	dbFullPath string
)

// Nonce 是 [NonceSize]byte 的别名.
type Nonce = [NonceSize]byte

// SecretKey 是 *[KeySize]byte 的别名.
type SecretKey = *[KeySize]byte

func init() {
	baseDir = getBaseDir()
	dbDirPath = filepath.Join(baseDir, DBDir)
	dbFullPath = filepath.Join(dbDirPath, DBName)
}

func getBaseDir() string {
	path, err := os.Executable()
	if err != nil {
		panic(err)
	}
	path, _ = filepath.EvalSymlinks(path)
	return filepath.Dir(path)
}

// NewFragmentName 返回一个新的数据库碎片文件名.
func NewFragmentName() string {
	name := strconv.FormatInt(time.Now().Unix(), 10)
	return name + FragExt
}
