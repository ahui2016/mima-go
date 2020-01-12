package main

import (
	"errors"
	"html/template"
	"os"
	"path/filepath"
)

// 一些常量
const (
	KeySize      = 32
	NonceSize    = 24
	passwordSize = 16

	listenAddr  = "127.0.0.1:80"
	dateAndTime = "2006-01-02 15:04:05"

	tmplDir = "tmpl"
	DBDir   = "mimadb"
	DBName  = "mima.mdb"
	TempDir = "temp_dir_for_test"

	// 数据库碎片文件的后缀名
	FragExt = ".mdb.frag"

	// 数据库备份文件的后缀名
	tarballExt = ".tar.gz"
)

var (
	baseDir     string
	dbDirPath   string
	dbFullPath  string
	tmplDirPath string
	templates   *template.Template

	mdb *MimaDB

	dbFileNotFound error
	errAliasExist  error
	errNeedTitle   error
)

// Nonce 是 [NonceSize]byte 的别名.
type Nonce = [NonceSize]byte

// SecretKey 是 *[KeySize]byte 的别名.
type SecretKey = *[KeySize]byte

func init() {
	baseDir = getBaseDir()
	dbDirPath = filepath.Join(baseDir, DBDir)
	dbFullPath = filepath.Join(dbDirPath, DBName)

	tmplDirPath = filepath.Join(baseDir, tmplDir)
	templates = template.Must(template.ParseGlob(filepath.Join(tmplDirPath, "*.html")))

	dbFileNotFound = errors.New("找不到数据库文件")
	errAliasExist = errors.New("该 alias 已存在")
	errNeedTitle = errors.New("Title 标题长度必须大于零")
}

func getBaseDir() string {
	path, err := os.Executable()
	if err != nil {
		panic(err) // 因为是在初始化阶段, 允许程序崩溃.
	}
	path, _ = filepath.EvalSymlinks(path)
	return filepath.Dir(path)
}
