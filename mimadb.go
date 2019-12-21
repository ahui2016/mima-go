package main

import (
	"container/list"
	"path/filepath"
	"sync"

	"github.com/ahui2016/mima-go/tarball"
	"github.com/ahui2016/mima-go/util"
)

// MimaDB 相当于一个数据表.
// 使用时注意需要利用 sync.RWMutex 上锁.
type MimaDB struct {
	sync.RWMutex

	// 主要用于 MimaDB.Add 中
	CurrentID uint

	// 原始数据, 按 UpdatedAt 排序.
	Items *list.List

	key SecretKey
}

// NewMimaDB 生成一个新的 MimaDB, 并对其中的 Items 进行初始化.
func NewMimaDB(key SecretKey) *MimaDB {
	if key == nil {
		panic("缺少key, 需要key")
	}
	items := new(MimaDB)
	items.key = key
	items.Items = list.New()
	return items
}

// Rebuild 读取数据库碎片, 整合到数据库文件中.
// 每次启动程序, 初始化时, 自动执行一次 Rebuild.
func (db *MimaDB) Rebuild() {
	dbMustExist()
	backupToTar()

	scanner := util.NewFileScanner(dbFullPath)
	for scanner.Scan() {

	}
}

// MakeFirstMima 生成第一条记录, 用于保存密码.
// 同时会生成数据库文件 mimadb/mima.db
func (db *MimaDB) MakeFirstMima() {
	dbMustNotExist()
	mima := NewMima("")
	// mima.ID = 0 默认为零
	mima.Notes = randomString()
	db.Add(mima)
	sealed := mima.Seal(db.key)
	writeFile(dbFullPath, sealed)
}

// GetByID 凭 id 找 mima, 如果找不到就返回 nil.
func (db *MimaDB) GetByID(id uint) *Mima {
	// 这里的算法效率不高, 当预估数据量较大时需要改用更高效率的算法.
	for e := db.Items.Front(); e != nil; e = e.Next() {
		mima := e.Value.(*Mima)
		if mima.ID == id {
			return mima
		}
	}
	return nil
}

// Add 新增一个 mima 到数据库中, 并生成一块数据库碎片.
func (db *MimaDB) Add(mima *Mima) {
	if db.Items.Len() == 0 {
		// 第一条记录特殊处理,
		// 尤其注意从数据库文件读取数据到内存时, 确保第一条读入的是那条特殊记录.
		db.Items.PushFront(mima)
		db.CurrentID++
		return
	}
	if len(mima.Title) == 0 {
		panic("Title 标题长度必须大于零")
	}
	mima.ID = db.CurrentID
	db.CurrentID++
	db.insertByUpdatedAt(mima)

	sealed := mima.Seal(db.key)
	writeFragFile(sealed)
}

// mustBeEmpty 确认内存中的数据库必须为空.
// 通常当不为空时不可进行初始化.
func (db *MimaDB) mustBeEmpty() {
	if db.Items.Len() != 0 {
		panic("初始化失败: 内存中的数据库已有数据")
	}
}

// InsertByUpdatedAt 把 mima 插入到适当的位置, 使链表保持有序.
func (db *MimaDB) insertByUpdatedAt(mima *Mima) {
	if e := db.findUpdatedBefore(mima); e != nil {
		db.Items.InsertBefore(mima, e)
	} else {
		db.Items.PushBack(mima)
	}
}

// findUpdatedBefore 寻找一条记录, 其更新日期早于参数 mima 的更新日期.
// 如果找不到则返回 nil, 表示参数 mima 的更新日期是最早的.
func (db *MimaDB) findUpdatedBefore(mima *Mima) *list.Element {
	for e := db.Items.Front(); e != nil; e = e.Next() {
		v := e.Value.(*Mima)
		if v.UpdatedAt <= mima.UpdatedAt {
			return e
		}
	}
	return nil
}

// backupToTar 把数据库文件以及碎片文件备份到一个 tarball 里.
// 主要在 Rebuild 之前使用, 以防万一 rebuild 出错.
// 为了方便测试返回 tarball 的完整路径.
func backupToTar() (filePath string) {
	files := filesToBackup()
	filePath = filepath.Join(dbDirPath, newBackupName())
	if err := tarball.Create(filePath, files); err != nil {
		panic(err)
	}
	return
}

// filesToBackup 返回需要备份的文件的完整路径.
func filesToBackup() []string {
	pattern := filepath.Join(dbDirPath, "*"+FragExt)
	filePaths, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}
	return append(filePaths, dbFullPath)
}
