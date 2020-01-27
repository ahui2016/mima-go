package util

import (
	"bufio"
	"fmt"
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

// NewFileScanner 打开指定文件并返回一个 Scanner, 以准备开始逐行读取文件内容.
func NewFileScanner(fullPath string) (*bufio.Scanner, *os.File, error) {
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, nil, err
	}
	return bufio.NewScanner(file), file, nil
}
