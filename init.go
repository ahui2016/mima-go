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
