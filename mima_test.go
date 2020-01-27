package main

import (
	"fmt"
	mimaDB "github.com/ahui2016/mima-go/db"
	"testing"
	"time"
)

// TestSortMima 用来测试 InsertByUpdatedAt 能否正确排序.
// 单独运行一个测试函数 go test -v -o ./mima.exe -run TestSortMima
// 或者 go test -v -o ./mima.exe github.com/ahui2016/mima-go -run TestSortMima
func TestSortMima(t *testing.T) {
	t.Skip("顺序测试在 mimadb_test.go 里做, 这里取消")
}

func TestMima_DeleteHistory(t *testing.T) {
	mima, err := mimaDB.NewMima("Mima for testing")
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
		if err := mima.DeleteHistory(wrongDatetime); err == nil {
			t.Fatal("want: Error Not Found, got: no error")
		}
	})
	for i := len(titles) - 1; i >= 0; i-- {
		datetime := mima.History[i].DateTime
		if err := mima.DeleteHistory(datetime); err != nil {
			t.Fatal(err)
		}
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
