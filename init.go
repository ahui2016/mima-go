package main

import (
	"os"
	"path/filepath"
)

// 一些常量
const (
	KeySize   = 32
	NonceSize = 24

	DBDir  = "mimadb"
	DBName = "mima.db"
)

var (
	baseDir    string
	dbDirPath  string
	dbFullPath string
)

// Nonce 是 [NonceSize]byte 的别名.
type Nonce [NonceSize]byte

// SecretKey 是 *[KeySize]byte 的别名.
type SecretKey *[KeySize]byte

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
