package util

import (
	"fmt"
	"io/ioutil"
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
