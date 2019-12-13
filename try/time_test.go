package tryandtest

import (
	"testing"
	"time"
)

func TestZero(t *testing.T) {
	zero := time.Unix(0, 0)
	t.Logf("Unix: %d", zero.Unix())
}

func TestNow(t *testing.T) {
	now := time.Now()
	t.Logf("Unix: %d", now.Unix())
	t.Logf("UnixNano: %d", now.UnixNano())
}
