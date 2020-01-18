package main

import (
	"crypto/sha256"
	"fmt"
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

func TestMima_DeleteHistory(t *testing.T) {
	mima, err := NewMima("Mima for testing")
	checkTestErr(t, err)

	titles := []string{"one", "two"}
	for _, v := range titles {
		// 确保两次生成历史记录之间超过 1 秒
		time.Sleep(1100 * time.Millisecond)
		t.Log("time.Sleep... (1.1 second)")
		t.Run("Test update from form", func(*testing.T) { // 注意这里的 *testing.T 没有变量名, 因为希望出错时使用上级的 t.Fatal 使该测试函数整体失败.
			needChangeIndex, needWriteFrag, err := mima.UpdateFromForm(&MimaForm{Title: v})
			checkTestErr(t, err)
			if !needChangeIndex || !needWriteFrag {
				t.Fatal("want 'needChangeIndex' and 'needWriteFrag' both true")
			}
		})
	}
	t.Run("Test count history items", func(*testing.T) {
		if len(mima.History) != len(titles) {
			t.Logf("len: %d, mima.History[0]: %v", len(mima.History), mima.History[0])
			t.Fatalf(
				"len(mima.History), want == %d, got != %d", len(titles), len(mima.History))
		}
	})
	t.Run("Test not found", func(*testing.T) {
		wrongDatetime := "abc"
		_, _, ok := mima.GetHistory(wrongDatetime)
		if ok {
			t.Fatal("want: ok == false, got: ok == true")
		}
	})
	for i := len(titles) - 1; i >= 0; i-- {
		datetime := mima.History[i].DateTime
		index, _, ok := mima.GetHistory(datetime)
		if !ok {
			t.Fatal("历史记录不存在:", index)
		}
		mima.DeleteHistory(index)
		name := fmt.Sprintf("Test deleting History[%d]", i)
		t.Run(name, func(*testing.T) {
			if len(mima.History) != i {
				t.Log(mima)
				t.Fatalf(
					"len(mima.History), want == %d, got != %d", i, len(mima.History))
			}
		})
	}
}

func checkTestErr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
