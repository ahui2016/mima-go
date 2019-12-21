package util

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
)

// WrapErrors 把多个错误合并为一个错误.
func WrapErrors(allErrors ...error) (wrapped error) {
	for i, err := range allErrors {
		if err != nil {
			wrapped = fmt.Errorf("%d: %w", i, err)
		}
	}
	return
}

// ReadFile 读取一个文件的全部内容.
func ReadFile(fullpath string) []byte {
	content, err := ioutil.ReadFile(fullpath)
	if err != nil {
		panic(err)
	}
	return content
}

// NewFileScanner 打开指定文件并返回一个 Scanner, 以准备开始逐行读取文件内容.
func NewFileScanner(fullpath string) *bufio.Scanner {
	file, err := os.Open(fullpath)
	if err != nil {
		panic(err)
	}
	return bufio.NewScanner(file)
}
