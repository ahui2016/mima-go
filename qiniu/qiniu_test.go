package qiniu

import "testing"

var qn = NewQiniu(
	"TvFfavKeGJbsSyOxMOcOweGCqyb4b_ghdbuUjYKL",
	"fPHLopOXEwyT2E62rK_6nPpRu0RbwU7seI8DMbae",
	"er0er0",
)

func TestQiniu_GetUpToken(t *testing.T) {
	t.Log(qn.GetUpToken())
}
