package tryandtest

import (
	"container/list"
	"testing"
)

type SliceInside struct {
	Slice []int
	List  *list.List
}

func NewSliceInSide() (s *SliceInside) {
	return
}

func TestSliceInStruct(t *testing.T) {
	s := new(SliceInside)
	// s := NewSliceInSide() // 不行, 需要初始化.
	t.Logf("%v", s.Slice)
	s.Slice = append(s.Slice, 3, 5)
	t.Logf("%v", s.Slice)

	s.List = list.New() // 实验证明需要先初始化
	s.List.PushBack("abc")
	t.Logf("%v", s.List.Back().Value)
}
