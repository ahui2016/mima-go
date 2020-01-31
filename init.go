package main

import (
	"errors"
	"flag"
	"fmt"
	mimaDB "github.com/ahui2016/mima-go/db"
	"html/template"
	"log"
	"os"
	"path/filepath"
)

// 一些常量
const (
	passwordSize = 16
	tmplDir = "tmpl"
	DBDir   = "mimadb"
	DBName  = "mima.db"
	TempDir = "temp_dir_for_test"
)

var (
	baseDir     string
	tmplDirPath string
	templates   *template.Template

	db *mimaDB.DB

	errMimaDeleted = errors.New("此记录已被删除")
)

var (
	localhost = "127.0.0.1"
	port = flag.Int("port", 10001, "80 <= port <= 65536")
)

type (
	Mima         = mimaDB.Mima
	Feedback     = mimaDB.Feedback
	MimaForm     = mimaDB.MimaForm
	SearchResult = mimaDB.SearchResult
)

func init() {
	baseDir = getBaseDir()
	dbDirPath := filepath.Join(baseDir, DBDir)
	dbFullPath := filepath.Join(dbDirPath, DBName)

	db = mimaDB.NewDB(dbFullPath, dbDirPath)

	tmplDirPath = filepath.Join(baseDir, tmplDir)
	templates = template.Must(template.ParseGlob(filepath.Join(tmplDirPath, "*.html")))
}

func getBaseDir() string {
	path, err := os.Executable()
	if err != nil {
		panic(err) // 因为是在初始化阶段, 允许程序崩溃.
	}
	path, _ = filepath.EvalSymlinks(path)
	return filepath.Dir(path)
}

func getAddr() string {
	if *port < 80 || *port > 65536 {
		log.Fatal("out of range: 80 <= port <= 65536")
	}
	return fmt.Sprintf("%s:%d", localhost, *port)
}
