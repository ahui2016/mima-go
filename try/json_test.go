package tryandtest

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/nacl/secretbox"
)

/*
 * go test -v github.com/ahui2016/mima-go/try -run TestUnMarshal
 */
func TestUnMarshal(t *testing.T) {
	key := sha256.Sum256([]byte("我是密码"))
	原始数据 := NewMima("我是标题")
	加密后的数据 := 原始数据.Seal(&key)

	临时文件, err := ioutil.TempFile("", "mima.*.db")
	if err != nil {
		panic(err)
	}
	defer os.Remove(临时文件.Name())
	// t.Log(临时文件.Name())

	if _, err := 临时文件.Write(加密后的数据); err != nil {
		panic(err)
	}
	if err := 临时文件.Close(); err != nil {
		panic(err)
	}

	文件内容, err := ioutil.ReadFile(临时文件.Name())
	if err != nil {
		panic(err)
	}

	解密后的数据, ok := DecryptMima(文件内容, &key)
	if !ok {
		panic("解密失败")
	}
	// t.Log(解密后的数据)

	if !解密后的数据.约等于(原始数据) {
		t.Error(解密后的数据)
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
	mima.Notes = randomString()
	mima.Nonce = newNonce()
	mima.CreatedAt = time.Now().Unix()
	mima.UpdatedAt = mima.CreatedAt
	return mima
}

func DecryptMima(box []byte, key *[32]byte) (*Mima, bool) {
	var nonce [24]byte
	copy(nonce[:], box[:24])

	blob, ok := secretbox.Open(nil, box[24:], &nonce, key)
	if !ok {
		return nil, ok
	}
	mima := new(Mima)
	if err := json.Unmarshal(blob, mima); err != nil {
		panic(err)
	}
	return mima, ok
}

func (mima *Mima) toJSON() []byte {
	blob, err := json.Marshal(mima)
	if err != nil {
		panic(err)
	}
	return blob
}

func (mima *Mima) Seal(key *[32]byte) []byte {
	return secretbox.Seal(mima.Nonce[:], mima.toJSON(), &mima.Nonce, key)
}

func (mima *Mima) 约等于(other *Mima) bool {
	if mima.Title == other.Title &&
		mima.Notes == other.Notes &&
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

func randomString() string {
	someBytes := make([]byte, 255)
	if _, err := rand.Read(someBytes); err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(someBytes)
}
