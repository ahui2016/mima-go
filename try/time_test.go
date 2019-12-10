package tryandtest

import (
	"testing"
	"time"
)

func TestZero(t *testing.T) {
	zero := time.Unix(0, 0)
	t.Errorf("Unix: %d", zero.Unix())
}

func TestNow(t *testing.T) {
	now := time.Now()
	t.Errorf("Unix: %d", now.Unix())
	t.Errorf("UnixNano: %d", now.UnixNano())
}
