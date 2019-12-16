package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// 一个 package 只能有一个 TestMain.
func TestMain(m *testing.M) {
	dbDirPath = filepath.Join(baseDir, TempDir)
	dbFullPath = filepath.Join(dbDirPath, DBName)
	app := m.Run()
	tempFiles, err := ioutil.ReadDir(dbDirPath)
	if err != nil {
		panic(err)
	}
	for _, file := range tempFiles {
		if err := os.Remove(filepath.Join(dbDirPath, file.Name())); err != nil {
			panic(err)
		}
	}
	os.Exit(app)
}

// 使用命令 go test -v -o ./mima.exe
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
	if dbDirPath != dbDirForTest {
		t.Errorf(dbDirPath)
	}
	if dbFullPath != dbPathForTest {
		t.Errorf(dbFullPath)
	}
}
