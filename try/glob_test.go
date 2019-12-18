package tryandtest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWD(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	pattern := filepath.Join(wd, "*"+".go")
	t.Log(pattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}
	t.Log(matches)
}
