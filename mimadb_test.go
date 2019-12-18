package main

import (
	"crypto/sha256"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"golang.org/x/crypto/nacl/secretbox"
)

// 使用命令 go test -v -o ./mima.exe
// 注意参数 -o, 用来强制指定文件夹, 如果不使用该参数, 测试有可能使用临时文件夹.
// 有时可能需要来回尝试 "./mima.exe" 或 "mima.exe".
func TestMakeFirstMima(t *testing.T) {
	if err := os.Remove(dbFullPath); err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	key := sha256.Sum256([]byte("我是密码"))
	testDB := NewMimaDB(&key)
	testDB.MakeFirstMima()

	// 检查内存中的 mima 是否正确
	if testDB.Items.Len() != 1 {
		t.Errorf("db.Items.Len() want: 1, got: %d", testDB.Items.Len())
	}
	mima := testDB.GetByID(0) // 第一条数据的 id 固定为零
	if mima == nil {
		t.Error("want a mima, got nil")
	}
	// t.Logf("len(mima.Notes) = %d", len(mima.Notes))
	if mima.Title != "" || len(mima.Notes) != 340 {
		t.Error("希望 Title 为空字符串, Notes 长度为 n, 但结果不是")
	}

	// 检查数据库文件中的 mima 是否正确
	if _, err := os.Stat(dbFullPath); os.IsNotExist(err) {
		t.Errorf("数据库文件 %s 应存在, 但结果不存在", dbFullPath)
	}
	解密后的数据 := readAndDecrypt(dbFullPath, &key)
	if !约等于(解密后的数据, mima) {
		t.Error("从数据库文件中恢复的 mima 与内存中的 mima 不一致")
	}
}

// 测试增加多条记录的情形
func TestAddMoreMimas(t *testing.T) {
	want := []*Mima{
		newRandomMima("二二二"),
		newRandomMima("六六六"),
	}

	if err := os.Remove(dbFullPath); err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	key := sha256.Sum256([]byte("我是密码"))
	testDB := NewMimaDB(&key)
	testDB.MakeFirstMima()

	for _, mima := range want {
		// 由于数据是按更新时间排序的, 为了使其有明显顺序, 因此明显地设置其更新时间.
		time.Sleep(100 * time.Millisecond)
		mima.UpdatedAt = time.Now().UnixNano()
		t.Logf("%s, %d", mima.Title, mima.UpdatedAt)
		testDB.Add(mima)
	}

	var got []*Mima
	mimaFiles, err := ioutil.ReadDir(dbDirPath)
	if err != nil {
		panic(err)
	}
	for _, f := range mimaFiles {
		fragmentPath := filepath.Join(dbDirPath, f.Name())
		mima := readAndDecrypt(fragmentPath, &key)
		if mima.ID != 0 {
			got = append(got, mima)
		}
	}
	sort.Slice(got, func(i, j int) bool {
		return got[i].UpdatedAt < got[j].UpdatedAt
	})

	for i := 0; i < len(want); i++ {
		if !约等于(want[i], got[i]) {
			t.Error("从数据库碎片文件中恢复的 mima 与内存中的 mima 不一致")
		}
	}
}

func newRandomMima(title string) *Mima {
	mima := NewMima(title)
	mima.Username = randomString()
	mima.Password = randomString()
	mima.Notes = randomString()
	return mima
}

func readAndDecrypt(fullpath string, key SecretKey) *Mima {
	box, err := ioutil.ReadFile(fullpath)
	if err != nil {
		panic(err)
	}
	return mustDecryptMima(box, key)
}

func decryptMima(box []byte, key SecretKey) (*Mima, bool) {
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

func mustDecryptMima(box []byte, key SecretKey) *Mima {
	mima, ok := decryptMima(box, key)
	if !ok {
		panic("解密失败")
	}
	return mima
}

func 约等于(mima, other *Mima) bool {
	if mima.Title == other.Title &&
		mima.Username == other.Username &&
		mima.Password == other.Password &&
		mima.Notes == other.Notes &&
		mima.Nonce == other.Nonce &&
		mima.CreatedAt == other.CreatedAt &&
		mima.UpdatedAt == other.UpdatedAt {
		return true
	}
	return false
}
