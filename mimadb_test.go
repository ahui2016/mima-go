package main

import (
	"crypto/sha256"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"golang.org/x/crypto/nacl/secretbox"
)

// 使用命令 go test -v -o ./mima.exe
// 注意参数 -o, 用来强制指定文件夹, 如果不使用该参数, 测试有可能使用临时文件夹.
// 有时可能需要来回尝试 "./mima.exe" 或 "mima.exe".
func TestMakeFirstMima(t *testing.T) {
	key := sha256.Sum256([]byte("我是密码"))
	db := NewMimaItems(&key)
	db.MakeFirstMima()

	// 检查内存中的 mima 是否正确
	if db.Items.Len() != 1 {
		t.Errorf("db.Items.Len() want: 1, got: %d", db.Items.Len())
	}
	mima := db.GetByID(0) // 第一条数据的 id 固定为零
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
	文件内容, err := ioutil.ReadFile(dbFullPath)
	if err != nil {
		panic(err)
	}
	解密后的数据, ok := decryptMima(文件内容, &key)
	if !ok {
		panic("解密失败")
	}
	if !约等于(解密后的数据, mima) {
		t.Error("从数据库文件中恢复的 mima 与内存中的 mima 不一致")
	}
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

func 约等于(mima, other *Mima) bool {
	if mima.Title == other.Title &&
		mima.Notes == other.Notes &&
		mima.Nonce == other.Nonce &&
		mima.CreatedAt == other.CreatedAt &&
		mima.UpdatedAt == other.UpdatedAt {
		return true
	}
	return false
}
