package main

import (
	"math/rand"
	"testing"
	"time"
)

// TestSortMima 用来测试 InsertByUpdatedAt 能否正确排序.
func TestSortMima(t *testing.T) {
	rand.Seed(42)
	hours := rand.Perm(24)

	mimaItems := NewMimaItems()
	for _, hour := range hours {
		mima := new(Mima)
		mima.UpdatedAt = time.Date(2019, time.May, 1, hour, 0, 0, 0, time.UTC).Unix()
		mimaItems.InsertByUpdatedAt(mima)
	}

	var got []int
	for e := mimaItems.Items.Front(); e != nil; e = e.Next() {
		mima := e.Value.(*Mima)
		updatedAt := time.Unix(mima.UpdatedAt, 0).UTC()
		got = append(got, updatedAt.Hour())
	}

	var want []int
	for i := 23; i >= 0; i-- {
		want = append(want, i)
	}

	for i, v := range got {
		if v != want[i] {
			t.Errorf("got[%d]: %d, want[%d]: %d", i, v, i, want[i])
		}
	}
}
