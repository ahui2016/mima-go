package tryandtest

import (
	"container/list"
	"testing"
)

type SliceInside struct {
	Slice []int
	List  *list.List
}

func TestSliceInStruct(t *testing.T) {
	s := new(SliceInside) // SliceInside 需要初始化, Slice 不需要.
	t.Logf("%v", s.Slice)
	s.Slice = append(s.Slice, 3, 5)
	t.Logf("%v", s.Slice)

	s.List = list.New() // 实验证明需要先初始化
	s.List.PushBack("abc")
	t.Logf("%v", s.List.Back().Value)
}
