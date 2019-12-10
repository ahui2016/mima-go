package main

import (
	"path/filepath"
	"testing"
)

// 使用命令 go test -v -o ./mima.exe
// 注意参数 -o, 用来强制指定文件夹, 如果不使用该参数, 测试有可能使用临时文件夹.
// 有时可能需要来回尝试 "./mima.exe" 或 "mima.exe".
// 具体的正确路径需要手动修改.
func TestPaths(t *testing.T) {
	if filepath.ToSlash(baseDir) != filepath.ToSlash("D:\\ComputerScience\\golang\\myprojects\\mima-go") {
		t.Errorf(baseDir)
	}
	if filepath.ToSlash(dbDirPath) != filepath.ToSlash("D:\\ComputerScience\\golang\\myprojects\\mima-go\\mimadb") {
		t.Errorf(dbDirPath)
	}
	if filepath.ToSlash(dbFullPath) != filepath.ToSlash("D:\\ComputerScience\\golang\\myprojects\\mima-go\\mimadb\\mima.db") {
		t.Errorf(dbFullPath)
	}
}
