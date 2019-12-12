package tryandtest

import (
	"crypto/rand"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestUnMarshal(t *testing.T) {
	mima := NewMima("hello")
	want, err := json.Marshal(mima)
	if err != nil {
		panic(err)
	}

	tmpFile, err := ioutil.TempFile("", "mima.*.db")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpFile.Name())
	// t.Error(tmpFile.Name())

	if _, err := tmpFile.Write(want); err != nil {
		panic(err)
	}
	if err := tmpFile.Close(); err != nil {
		panic(err)
	}

	content, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		panic(err)
	}

	got := new(Mima)
	if err := json.Unmarshal(content, got); err != nil {
		panic(err)
	}

	if !got.约等于(mima) {
		t.Error(got)
	}
}

type Mima struct {
	Title        string
	Alias        string
	Nonce        [24]byte
	Username     string
	Password     string
	Notes        string
	Favorite     bool
	CreatedAt    int64
	UpdatedAt    int64
	DeletedAt    int64
	HistoryItems []History
}

type History struct {
	Title     string
	Username  string
	Password  string
	Notes     string
	UpdatedAt int64
}

func NewMima(title string) *Mima {
	mima := new(Mima)
	mima.Title = title
	mima.Nonce = newNonce()
	mima.CreatedAt = time.Now().Unix()
	mima.UpdatedAt = mima.CreatedAt
	return mima
}

func (mima *Mima) 约等于(other *Mima) bool {
	if mima.Title == other.Title &&
		mima.Nonce == other.Nonce &&
		mima.CreatedAt == other.CreatedAt &&
		mima.UpdatedAt == other.UpdatedAt {
		return true
	}
	return false
}

func newNonce() (nonce [24]byte) {
	if _, err := rand.Read(nonce[:]); err != nil {
		panic(err)
	}
	return
}
