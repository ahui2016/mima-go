package main

// 使用命令 go test -v -o ./mima.exe
// 注意参数 -o, 用来强制指定文件夹, 如果不使用该参数, 测试有可能使用临时文件夹.
// 有时可能需要来回尝试 "./mima.exe" 或 "mima.exe".

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ahui2016/mima-go/tarball"
)

// 用于在测试之前删除数据库文件 (dbFullPath in temp_dir_for_test)
func removeDB() error {
	if err := os.Remove(dbFullPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func TestMakeFirstMima(t *testing.T) {
	key := sha256.Sum256([]byte("我是密码"))
	testDB := NewMimaDB(&key)

	if err := removeDB(); err != nil {
		t.Fatal(err)
	}
	testDB.MakeFirstMima()

	// 检查内存中的 mima 是否正确
	if testDB.Items.Len() != 1 {
		t.Fatalf("db.Items.Len() want: 1, got: %d", testDB.Items.Len())
	}
	mima := testDB.GetByID(0) // 第一条数据的 id 固定为零
	if mima == nil {
		t.Fatal("want a mima, got nil")
	}
	// t.Logf("len(mima.Notes) = %d", len(mima.Notes))
	if mima.Title != "" || len(mima.Notes) != 340 {
		t.Fatal("希望 Title 为空字符串, Notes 长度为 n, 但结果不是")
	}

	// 检查数据库文件中的 mima 是否正确
	if _, err := os.Stat(dbFullPath); os.IsNotExist(err) {
		t.Fatalf("数据库文件 %s 应存在, 但结果不存在", dbFullPath)
	}
	解密后的数据, err := readAndDecrypt(dbFullPath, &key)
	if err != nil {
		t.Fatalf("%w: %s", err, dbFullPath)
	}
	if !约等于(解密后的数据, mima) {
		t.Fatal("从数据库文件中恢复的 mima 与内存中的 mima 不一致")
	}
}

// 测试增加多条记录的情形
func TestAddMoreMimas(t *testing.T) {
	want := []*Mima{
		newRandomMima("鹅鹅鹅"),
		newRandomMima("二二二"),
		newRandomMima("六六六"),
	}
	key := sha256.Sum256([]byte("我是密码"))
	testDB := NewMimaDB(&key)
	if err := removeDB(); err != nil {
		t.Fatal(err)
	}
	testDB.MakeFirstMima()

	for _, mima := range want {
		// 由于数据是按更新时间排序的, 为了使其有明显顺序, 因此明显地设置其更新时间.
		time.Sleep(100 * time.Millisecond)
		mima.UpdatedAt = time.Now().UnixNano()
		testDB.Add(mima)
	}

	filePaths, err := fragFilePaths()
	if err != nil {
		t.Fatal(err)
	}
	// TestFragFilePaths 测试所获取的数据库碎片文件路径是否升序排列.
	t.Run("TestFragFilePaths", func(t *testing.T) {
		for i := 0; i < len(filePaths)-1; i++ {
			if filePaths[i] >= filePaths[i+1] {
				t.Errorf("第 %d 个大于或等于第 %d 个", i, i+1)
			}
		}
	})
	if t.Failed() {
		t.FailNow()
	}

	var got []*Mima
	for _, f := range filePaths {
		mima, err := readAndDecrypt(f, &key)
		if err != nil {
			t.Fatalf("%w: %s", err, f)
		}
		got = append(got, mima)
	}

	for i := 0; i < len(want); i++ {
		if !约等于(want[i], got[i]) {
			t.Fatal("从数据库碎片文件中恢复的 mima 与内存中的 mima 不一致")
		}
	}

	// 由于刚好需要用到这里 "母测试" 产生的文件, 因此在此添加 "子测试".
	// 测试备份是否成功 (检查备份文件 tarball 的内容).
	t.Run("TestBackup", func(t *testing.T) {
		var (
			sumOfOrigins [][]byte // 原始文件的 checksum
			sumOfBackups [][]byte // 备份文件的 checksum
		)
		tarFilePath, err := backupToTar()
		if err != nil {
			t.Fatal(err)
		}
		tarballReader, err := tarball.NewReader(tarFilePath)
		if err != nil {
			t.Fatal(err)
		}
		sumOfBackups, err = tarballReader.Sha512()
		if err != nil {
			t.Fatal(err)
		}
		if err := tarballReader.Close(); err != nil {
			t.Fatal(err)
		}

		files, err := filesToBackup()
		if err != nil {
			t.Fatal(err)
		}
		sumOfOrigins = getChecksums(files)

		for i := 0; i < len(sumOfOrigins); i++ {
			if !bytes.Equal(sumOfOrigins[i], sumOfBackups[i]) {
				t.Errorf("第 %d 个文件的备份与原文件的 checksum 不一致", i+1)
			}
		}
	})
}

// getChecksums 返回 files(完整路径) 的 SHA512 checksum.
func getChecksums(files []string) (checksums [][]byte) {
	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			panic(err)
		}
		sum := sha512.Sum512(content)
		checksums = append(checksums, sum[:])
	}
	return
}

func newRandomMima(title string) *Mima {
	mima, err := NewMima(title)
	if err != nil {
		panic(err)
	}
	mima.Username = randomString()
	mima.Password = randomString()
	mima.Notes = randomString()
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
