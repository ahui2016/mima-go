package tryandtest

import (
	"crypto/rand"
	"math/big"
	"strconv"
	"testing"
	"time"
)

func TestNewID(t *testing.T) {
	allID := make(map[string]bool)
	for i := 0; i < 10000; i++ {
		id, _ := NewID()
		if allID[id] {
			t.Fatalf("第 %d 个 id (%s) 发生碰撞", i, id)
		}
		allID[id] = true
	}
}

func NewID() (id string, err error) {
	var x int64 = 100_000_000
	n, err := rand.Int(rand.Reader, big.NewInt(x))
	if err != nil {
		return
	}
	timestamp := time.Now().Unix()
	id64 := timestamp * x + n.Int64()
	id = strconv.FormatInt(id64, 36)
	return
}
