package main

// 使用命令 go test -v -o ./mima.exe
// 注意参数 -o, 用来强制指定文件夹, 如果不使用该参数, 测试有可能使用临时文件夹.
// 有时可能需要来回尝试 "./mima.exe" 或 "mima.exe".

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/ahui2016/mima-go/tarball"
)

// 用于在测试之前删除旧的数据库文件 (dbFullPath in temp_dir_for_test)
func removeDB() error {
	if err := os.Remove(dbFullPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

//noinspection NonAsciiCharacters
func TestMakeFirstMima(t *testing.T) {
	key := sha256.Sum256([]byte("我是密码"))
	testDB := NewMimaDB(&key)
	testDB.Lock()
	defer testDB.Unlock()

	if err := removeDB(); err != nil {
		t.Fatal(err)
	}
	if err := testDB.MakeFirstMima(); err != nil {
		t.Fatal(err)
	}

	// 检查内存中的 mima 是否正确
	if testDB.Len() != 1 {
		t.Fatalf("mdb.Items.Len() want: 1, got: %d", testDB.Len())
	}
	mima := testDB.Items[0]
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
		t.Fatalf("%v: %s", err, dbFullPath)
	}
	if !约等于(解密后的数据, mima) {
		t.Fatal("从数据库文件中恢复的 mima 与内存中的 mima 不一致")
	}
}

// 测试增加多条记录的情形
// 需要小心处理 sync.RWMutex 的锁
func TestAddMoreMimas(t *testing.T) {
	key := sha256.Sum256([]byte("我是密码"))
	testDB := NewMimaDB(&key)

	if err := removeDB(); err != nil {
		t.Fatal(err)
	}

	testDB.Lock()
	if err := testDB.MakeFirstMima(); err != nil {
		t.Fatal(err)
	}
	testDB.Unlock()

	titles := []string{"鹅鹅鹅", "二二二", "六六六"}
	want := make([]*Mima, len(titles))
	for i, title := range titles {
		// 由于数据是按更新时间排序的, 为了使其有明显顺序, 因此明显地使其具有时间间隔.
		time.Sleep(100 * time.Millisecond)
		mima, err := newRandomMima(title)
		if err != nil {
			t.Fatal(err)
		}
		want[i] = mima

		testDB.Lock()
		if err := testDB.Add(mima); err != nil {
			t.Fatal(err)
		}
		testDB.Unlock()
	}

	fragFiles, err := fragFilePaths()
	if err != nil {
		t.Fatal(err)
	}
	// TestFragFilePaths 测试所获取的数据库碎片文件路径是否升序排列.
	t.Run("TestFragFilePaths", func(t *testing.T) {
		for i := 0; i < len(fragFiles)-1; i++ {
			if fragFiles[i] >= fragFiles[i+1] {
				t.Fatalf("第 %d 个大于或等于第 %d 个", i+1, i+2)
			}
		}
	})

	var got []*Mima
	for _, f := range fragFiles {
		mima, err := readAndDecrypt(f, &key)
		if err != nil {
			t.Fatalf("%v: %s", err, f)
		}
		got = append(got, mima)
	}

	if got == nil {
		t.Fatal()
	}
	for i := 0; i < len(want); i++ {
		if !约等于(want[i], got[i]) {
			t.Fatal("从数据库碎片文件中恢复的 mima 与内存中的 mima 不一致")
		}
	}

	// TestOrderOfDB 确认内存数据库中的条目按升序排列 (老数据在前, 新数据在后).
	t.Run("TestOrderOfDB", func(t *testing.T) {
		testDB.RLock()
		mimaSlice := testDB.Items
		testDB.RUnlock()

		for i := 0; i < len(mimaSlice)-1; i++ {
			a := mimaSlice[i]
			b := mimaSlice[i+1]
			if a.UpdatedAt >= b.UpdatedAt {
				t.Logf("id: %d, Title: %s, id: %d, Title: %s", a.ID, a.Title, b.ID, b.Title)
				t.Logf("a: %s, b: %s", time.Unix(0, a.UpdatedAt), time.Unix(0, b.UpdatedAt))
				t.Fatalf("第 %d 个元素的更新日期大于或等于第 %d 个", i+1, i+2)
			}
		}
	})

	// 由于刚好需要用到这里 "母测试" 产生的文件, 因此在此添加 "子测试".
	// 测试备份是否成功 (检查备份文件 tarball 的内容).
	t.Run("TestBackup", func(t *testing.T) {
		var (
			sumOfOrigins [][]byte // 原始文件的 checksum
			sumOfBackups [][]byte // 备份文件的 checksum
		)
		allBackupFiles := filesToBackup(fragFiles)
		tarFilePath, err := backupToTar(allBackupFiles)
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
		sumOfOrigins, err = getChecksums(allBackupFiles)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < len(sumOfOrigins); i++ {
			if !bytes.Equal(sumOfOrigins[i], sumOfBackups[i]) {
				t.Errorf("第 %d 个文件的备份与原文件的 checksum 不一致", i+1)
			}
		}
	})

	// 由于刚好需要用到这里 "母测试" 产生的文件, 因此在此添加 "子测试".
	// 测试根据数据库文件和碎片文件重建内存数据库的功能.
	var (
		tarballFile string
		rebuiltDB   *MimaDB
	)

	t.Run("TestGetRebuiltDB", func(t *testing.T) {
		var err error
		rebuiltDB = NewMimaDB(&key)

		rebuiltDB.Lock()
		if tarballFile, err = rebuiltDB.Rebuild(); err != nil {
			t.Fatal(err)
		}
		rebuiltDB.Unlock()
	})

	t.Run("TestTarballExist", func(t *testing.T) {
		if _, err := os.Stat(tarballFile); os.IsNotExist(err) {
			t.Fatalf("找不到备份文件: %s", tarballFile)
		}
	})

	testDB.RLock()
	rebuiltDB.RLock()
	t.Run("TestReBuiltDB", func(t *testing.T) {
		if !mimaSliceEqual(testDB.Items, rebuiltDB.Items) {
			t.Fatal("恢复的内存数据库与原数据库不一致")
		}
	})
	rebuiltDB.RUnlock()
	testDB.RUnlock()

	// 在 Rebuild 过程中会删除数据库碎片文件, 在这里检查是否已删除.
	t.Run("TestFragFileShouldNotExist", func(t *testing.T) {
		fragFiles, err := fragFilePaths()
		if err != nil {
			t.Fatal(err)
		}
		if fragFiles != nil {
			t.Fatal("数据库碎片文件应不存在, 但存在.")
		}
	})

	// 由于数据库文件已被重写, 因此重新读取数据库文件的内容, 验证其正确性.
	t.Run("TestRewriteDBFile", func(t *testing.T) {
		restoredDB := NewMimaDB(&key)

		restoredDB.Lock()
		if err := restoredDB.scanDBtoMemory(); err != nil {
			t.Fatal(err)
		}
		restoredDB.Unlock()

		testDB.RLock()
		restoredDB.RLock()
		if !mimaSliceEqual(testDB.Items, restoredDB.Items) {
			t.Fatal("数据库文件的内容与原数据库不一致")
		}
		restoredDB.RUnlock()
		testDB.RUnlock()
	})
}

// mimaSliceEqual 判断两个元素类型为 *Mima 的 list 是否相等.
func mimaSliceEqual(a, b []*Mima) bool {
	if len(a) != len(b) {
		log.Println("两个 slice 的长度不一致")
		return false
	}
	for i := 0; i < len(a); i++ {
		m1 := a[i]
		m2 := b[i]
		if !约等于(m1, m2) {
			log.Printf("%s 不等于 %s", m1.Title, m2.Title)
			return false
		}
	}
	return true
}

// getChecksums 返回 files(完整路径) 的 SHA512 checksum.
func getChecksums(files []string) (checksums [][]byte, err error) {
	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}
		sum := sha512.Sum512(content)
		checksums = append(checksums, sum[:])
	}
	return
}

func newRandomMima(title string) (*Mima, error) {
	mima, err := NewMima(title)
	if err != nil {
		return nil, err
	}
	mima.Username = randomString()
	mima.Password = randomString()
	mima.Notes = randomString()
	return mima, nil
}

//noinspection NonAsciiCharacters
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
