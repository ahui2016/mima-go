package main

import (
	"errors"
	"os"
	"path/filepath"
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

	// 数据库备份文件的后缀名
	tarballExt = ".tar.gz"
)

var (
	baseDir    string
	dbDirPath  string
	dbFullPath string

	dbFileNotFound error
)

// Nonce 是 [NonceSize]byte 的别名.
type Nonce = [NonceSize]byte

// SecretKey 是 *[KeySize]byte 的别名.
type SecretKey = *[KeySize]byte

func init() {
	baseDir = getBaseDir()
	dbDirPath = filepath.Join(baseDir, DBDir)
	dbFullPath = filepath.Join(dbDirPath, DBName)
	dbFileNotFound = errors.New("找不到数据库文件")
}

func getBaseDir() string {
	path, err := os.Executable()
	if err != nil {
		panic(err) // 因为是在初始化阶段, 允许程序崩溃.
	}
	path, _ = filepath.EvalSymlinks(path)
	return filepath.Dir(path)
}
