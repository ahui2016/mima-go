package main

import (
	"path/filepath"
	"testing"
)

// 使用命令 go test -v -o ./mima.exe
// 注意参数 -o, 用来强制指定文件夹, 如果不使用该参数, 测试有可能使用临时文件夹.
// 有时可能需要来回尝试 "./mima.exe" 或 "mima.exe".
func TestPaths(t *testing.T) {
	// 具体的 workingDir 需要手动修改.
	workingDir := "D:\\ComputerScience\\golang\\myprojects\\mima-go"
	// os.Chdir(workingDir)

	if baseDir != workingDir {
		t.Errorf(baseDir)
	}
	if dbDirPath != filepath.Join(workingDir, DBDir) {
		t.Errorf(dbDirPath)
	}
	if dbFullPath != filepath.Join(workingDir, DBDir, DBName) {
		t.Errorf(dbFullPath)
	}
}
