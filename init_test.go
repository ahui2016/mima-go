package main

import (
	mimaDB "github.com/ahui2016/mima-go/db"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
)

// 一个 package 只能有一个 TestMain.
func TestMain(m *testing.M) {
	log.Println("Ding!")
	dbDirPath := filepath.Join(baseDir, TempDir)
	dbFullPath := filepath.Join(dbDirPath, DBName)
	db = mimaDB.NewDB(dbFullPath, dbDirPath)
	app := m.Run()
	tempFiles, err := ioutil.ReadDir(dbDirPath)
	if err != nil {
		panic(err)
	}
	for _, file := range tempFiles {
		tempFile := filepath.Join(dbDirPath, file.Name())
		if err := os.Remove(tempFile); err != nil {
			panic(err)
		}
		log.Printf("已删除 %s", tempFile)
	}
	log.Println("Dong!")
	os.Exit(app)
}

// 使用命令 go test -v -o mima.exe
// 注意参数 -o, 用来强制指定文件夹, 如果不使用该参数, 测试有可能使用临时文件夹.
// 我这里出现了奇怪的情况, 有时该命令无法更改文件夹, 需要尝试多次才能成功.
func TestPaths(t *testing.T) {
	// 具体的 workingDir 需要手动修改.
	exeDir := "D:\\ComputerScience\\golang\\myprojects\\mima-go"
	// os.Chdir(workingDir)
	dbDirForTest := filepath.Join(exeDir, TempDir)
	dbPathForTest := filepath.Join(dbDirForTest, DBName)

	if baseDir != exeDir {
		t.Errorf(baseDir)
	}
	if db.BackupDir != dbDirForTest {
		t.Errorf(db.BackupDir)
	}
	if db.FullPath != dbPathForTest {
		t.Errorf(db.FullPath)
	}
}
