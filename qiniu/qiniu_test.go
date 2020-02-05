package qiniu

import (
	"github.com/qiniu/api.v7/v7/storage"
	"testing"
)

var qn = NewQiniu(
	"TvFfavKeGJbsSyOxMOcOweGCqyb4b_ghdbuUjYKL",
	"fPHLopOXEwyT2E62rK_6nPpRu0RbwU7seI8DMbae",
	"er0er0",
	"mima-go/temp",
	&storage.ZoneHuanan,
)

func TestQiniu_createUpToken(t *testing.T) {
	t.Skip("一般情况下在 TestQiniu_Upload 中已包含本测试")
	qn.createUpToken()
	t.Log(qn.upToken)
}

func TestQiniu_Upload(t *testing.T) {
	for _, f := range []string{"qiniu_test.go", "qiniu.go", "qiniu.go", "qiniu.go", "abcd"} {
		ret, err := qn.Upload(f, true)
		if err != nil {
			t.Log(err, f)
		}
		t.Log(ret)
	}
}
