package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	mimaDB "github.com/ahui2016/mima-go/db"
	"github.com/ahui2016/mima-go/ibm"
	"html/template"
	"log"
	"os"
	"path/filepath"
)

// 一些常量
const (
	passwordSize = 16
	tmplDir      = "tmpl"
	DBDir        = "mimadb"
	DBName       = "mima.db"
	TempDir      = "temp_dir_for_test"
)

var (
	baseDir     string
	tmplDirPath string
	templates   *template.Template

	db  *mimaDB.DB
	cos *ibm.COS

	errMimaDeleted = errors.New("此记录已被删除")
)

var (
	localhost = "127.0.0.1"
	port      = flag.Int("port", 10001, "端口: 80 <= port <= 65536")
	validTerm = flag.Int("term", 30, "有效期: 1 <= term(minutes) <= 1024")
)

type (
	Mima     = mimaDB.Mima
	MimaForm = mimaDB.MimaForm
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

func getTerm() int {
	if *validTerm < 1 || *validTerm > 1024 {
		log.Fatal("out of range: 1 <= term(minutes) <= 1024")
	}
	return *validTerm
}

// makeCOS 生成一个 COS 并保存到全局变量 cos 中.
func makeCOS(settings64 string) error {
	settings, err := NewSettingsFromJSON64(settings64)
	if err != nil {
		return err
	}
	cos = newCOSFromSettings(settings)
	return nil
}

func newCOSFromSettings(settings *Settings) *ibm.COS {
	return ibm.NewCOS(settings.ApiKey, settings.ServiceInstanceID, settings.ServiceEndpoint,
		settings.BucketLocation, settings.BucketName, settings.ObjKeyPrefix)
}

func updateSettings(settings Settings) error {
	settingsJson, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	settings64 := base64.StdEncoding.EncodeToString(settingsJson)
	return db.UpdateSettings(settings64)
}
