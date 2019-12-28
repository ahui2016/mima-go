package main

import (
	"bufio"
	"container/list"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/ahui2016/mima-go/util"
)

// MimaDB 相当于一个数据表.
// 使用时注意需要利用 sync.RWMutex 上锁.
type MimaDB struct {
	sync.RWMutex

	// 主要用于 MimaDB.Add 中
	CurrentID int

	// 原始数据, 按 UpdatedAt 排序.
	Items *list.List

	key SecretKey
}

// NewMimaDB 生成一个新的 MimaDB, 并对其中的 Items 进行初始化.
func NewMimaDB(key SecretKey) *MimaDB {
	if key == nil {
		// 因为改错误属于 "编程时" 错误, 不是 "运行时" 错误, 可在运行前处理,
		// 因此不返回错误信息, 而是让程序直接崩溃.
		panic("缺少key, 需要key")
	}
	items := new(MimaDB)
	items.key = key
	items.Items = list.New()
	return items
}

// Rebuild 填充内存数据库，读取数据库碎片, 整合到数据库文件中.
// 每次启动程序, 初始化时, 如果已有账号, 自动执行一次 Rebuild.
// 为了方便测试返回 tarball 文件路径.
func (db *MimaDB) Rebuild() (tarballFile string, err error) {
	var (
		fragFiles []string
	)
	if !db.isEmpty() {
		return tarballFile, errors.New("初始化失败: 内存中的数据库已有数据")
	}
	if dbFileIsNotExist() {
		return tarballFile, dbFileNotFound
	}
	if fragFiles, err = fragFilePaths(); err != nil {
		return
	}
	if tarballFile, err = backupToTar(filesToBackup(fragFiles)); err != nil {
		return
	}
	if err = db.scanDBtoMemory(); err != nil {
		return
	}
	if err = db.readFragFilesAndUpdate(fragFiles); err != nil {
		return
	}
	if err = db.rewriteDBFile(); err != nil {
		return
	}
	err = db.deleteFragFiles(fragFiles)
	return
}

// ToSlice 把 list 转换为 slice, 保持其中元素的顺序.
func (db *MimaDB) ToSlice() (mimaSlice []*Mima) {
	for e := db.Items.Front(); e != nil; e = e.Next() {
		mima := e.Value.(*Mima)
		mimaSlice = append(mimaSlice, mima)
	}
	return
}

func (db *MimaDB) deleteFragFiles(filePaths []string) error {
	for _, f := range filePaths {
		if err := os.Remove(f); err != nil {
			return err
		}
		log.Printf("已删除 %s", f)
	}
	return nil
}

// scanDBtoMemory 读取 dbFullPath, 填充 MimaDB.
func (db *MimaDB) scanDBtoMemory() error {
	scanner, file, err := util.NewFileScanner(dbFullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	for scanner.Scan() {
		box64 := scanner.Text()
		mima, err := DecryptToMima(box64, db.key)
		if err != nil {
			return fmt.Errorf("在初始化阶段解密失败: %w", err)
		}
		mima.ID = db.CurrentID
		db.CurrentID++
		db.Items.PushBack(mima)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// readFragFilesAndUpdate 读取数据库碎片文件, 并根据其内容更新内存数据库.
// 分为 新增, 更新, 软删除, 彻底删除 四种情形.
func (db *MimaDB) readFragFilesAndUpdate(filePaths []string) error {
	for _, f := range filePaths {
		mima, err := readAndDecrypt(f, db.key)
		if err != nil {
			return err
		}
		if mima.UpdatedAt == mima.CreatedAt {
			// 新增
			db.insertByUpdatedAt(mima)
			continue
		}

		item := db.GetByID(mima.ID)
		if item == nil {
			return fmt.Errorf("NotFound: 找不到 id: %d 的条目", mima.ID)
		}

		if mima.UpdatedAt > mima.CreatedAt {
			// 更新
			item.Update(mima)
			continue
		}
		if mima.DeletedAt > 0 {
			// 软删除
			item.Delete()
			continue
		}
		if mima.UpdatedAt == 0 {
			// 彻底删除
			if err := db.DeleteByID(mima.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// rewriteDBFile 覆盖重写数据库文件, 将其更新为当前内存数据库的内容.
func (db *MimaDB) rewriteDBFile() error {
	dbFile, err := os.Create(dbFullPath)
	if err != nil {
		return err
	}
	defer dbFile.Close()

	dbWriter := bufio.NewWriter(dbFile)
	for e := db.Items.Front(); e != nil; e = e.Next() {
		mima := e.Value.(*Mima)
		box64, err := mima.Seal(db.key)
		if err != nil {
			return err
		}
		if err := bufWriteln(dbWriter, box64); err != nil {
			return err
		}
	}
	if err := dbWriter.Flush(); err != nil {
		return err
	}
	return nil
}

// MakeFirstMima 生成第一条记录, 用于保存密码.
// 同时会生成数据库文件 mimadb/mima.db
func (db *MimaDB) MakeFirstMima() error {
	if !dbFileIsNotExist() {
		return errors.New("数据库文件已存在, 不可重复创建")
	}
	mima, err := NewMima("")
	if err != nil {
		return err
	}
	// mima.ID = 0 默认为零
	mima.Notes = randomString()
	db.Add(mima)
	box64, err := mima.Seal(db.key)
	if err != nil {
		return err
	}
	return writeFile(dbFullPath, box64)
}

// GetByID 凭 id 找 mima, 如果找不到就返回 nil.
func (db *MimaDB) GetByID(id int) *Mima {
	if e := db.getElementByID(id); e != nil {
		return e.Value.(*Mima)
	}
	return nil
}

func (db *MimaDB) getElementByID(id int) *list.Element {
	// 这里的算法效率不高, 当预估数据量较大时需要改用更高效率的算法.
	for e := db.Items.Front(); e != nil; e = e.Next() {
		mima := e.Value.(*Mima)
		log.Printf("getElementByID, id: %d, Title: %s", mima.ID, mima.Title)
		if mima.ID == id {
			return e
		}
	}
	return nil
}

// Add 新增一个 mima 到数据库中, 并生成一块数据库碎片.
func (db *MimaDB) Add(mima *Mima) error {
	if db.Items.Len() == 0 {
		// 第一条记录特殊处理,
		// 尤其注意从数据库文件读取数据到内存时, 确保第一条读入的是那条特殊记录.
		db.Items.PushFront(mima)
		db.CurrentID++
		return nil
	}
	if len(mima.Title) == 0 {
		return errors.New("Title 标题长度必须大于零")
	}
	mima.ID = db.CurrentID
	db.CurrentID++
	db.insertByUpdatedAt(mima)

	sealed, err := mima.Seal(db.key)
	if err != nil {
		return err
	}
	return writeFragFile(sealed)
}

// DeleteByID 删除内存数据库中的指定条目.
func (db *MimaDB) DeleteByID(id int) error {
	e := db.getElementByID(id)
	if e == nil {
		return fmt.Errorf("NotFound: 找不到 id: %d 的条目", id)
	}
	db.Items.Remove(e)
	return nil
}

// mustBeEmpty 确认内存中的数据库必须为空.
// 通常当不为空时不可进行初始化.
func (db *MimaDB) isEmpty() bool {
	if db.Items.Len() == 0 {
		return true
	}
	return false
}

// InsertByUpdatedAt 把 mima 插入到适当的位置, 使链表保持有序.
func (db *MimaDB) insertByUpdatedAt(mima *Mima) {
	if e := db.findUpdatedAfter(mima); e != nil {
		db.Items.InsertAfter(mima, e)
	} else {
		db.Items.PushBack(mima)
	}
}

// findUpdatedAfter 寻找一条记录, 其更新日期大于(晚于)参数 mima 的更新日期.
// 如果找不到则返回 nil, 表示参数 mima 的更新日期是最晚(最近)的.
func (db *MimaDB) findUpdatedAfter(mima *Mima) *list.Element {
	for e := db.Items.Front(); e != nil; e = e.Next() {
		v := e.Value.(*Mima)
		if v.UpdatedAt >= mima.UpdatedAt {
			return e
		}
	}
	return nil
}
