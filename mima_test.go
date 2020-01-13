package main

import (
	"crypto/sha256"
	"math/rand"
	"testing"
	"time"
)

// TestSortMima 用来测试 InsertByUpdatedAt 能否正确排序.
// 单独运行一个测试函数 go test -v -o ./mima.exe -run TestSortMima
// 或者 go test -v -o ./mima.exe github.com/ahui2016/mima-go -run TestSortMima
func TestSortMima(t *testing.T) {
	t.Skip("顺序测试在 mimadb_test.go 里做, 这里取消")
	rand.Seed(42)
	hours := rand.Perm(24)

	key := sha256.Sum256([]byte("我是密码"))
	db := NewMimaDB(&key)
	db.Lock()
	db.Unlock()

	for _, hour := range hours {
		mima := new(Mima)
		mima.UpdatedAt = time.Date(2019, time.May, 1, hour, 0, 0, 0, time.UTC).UnixNano()
		db.Items = append(db.Items, mima)
	}

	var got []int
	for _, mima := range db.Items {
		updatedAt := time.Unix(0, mima.UpdatedAt).UTC()
		got = append(got, updatedAt.Hour())
	}

	var want = make([]int, 24)
	for i := 23; i >= 0; i-- {
		want[23-i] = i
	}

	for i, v := range got {
		if v != want[i] {
			t.Fatalf("got[%d]: %d, want[%d]: %d", i, v, i, want[i])
		}
	}
}
