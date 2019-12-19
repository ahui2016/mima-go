package util

import "fmt"

// wrapErrors 把多个错误合并为一个错误.
func WrapErrors(allErrors ...error) (wrapped error) {
	for i, err := range allErrors {
		if err != nil {
			wrapped = fmt.Errorf("%d: %w", i, err)
		}
	}
	return
}
