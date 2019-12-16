package main

import (
	"container/list"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// MimaItems 相当于一个数据表.
// 使用时注意需要利用 sync.RWMutex 上锁.
type MimaItems struct {
	sync.RWMutex

	// 主要用于 MimaItems.Add 中
	CurrentID uint

	// 原始数据, 按 UpdatedAt 排序.
	Items *list.List

	key SecretKey
}

// NewMimaItems 生成一个新的 MimaItems, 并对其中的 Items 进行初始化.
func NewMimaItems(key SecretKey) *MimaItems {
	if key == nil {
		panic("缺少key, 需要key")
	}
	items := new(MimaItems)
	items.key = key
	items.Items = list.New()
	return items
}

// Rebuild 读取数据库碎片, 整合到数据库文件中.
// 每次启动程序, 初始化时, 自动执行一次 Rebuild.
func (db *MimaItems) Rebuild() {}

// MakeFirstMima 生成第一条记录, 用于保存密码.
func (db *MimaItems) MakeFirstMima() {
	if _, err := os.Stat(dbFullPath); os.IsExist(err) {
		panic("数据库文件已存在, 不可重复创建")
	}
	mima := NewMima("")
	// mima.ID = 0 默认为零
	mima.Notes = randomString()
	db.Add(mima)
	sealed := mima.Seal(db.key)
	if err := ioutil.WriteFile(dbFullPath, sealed, 0644); err != nil {
		panic(err)
	}
}

// GetByID 凭 id 找 mima, 如果找不到就返回 nil.
func (db *MimaItems) GetByID(id uint) *Mima {
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
func (db *MimaItems) Add(mima *Mima) {
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
	fragmentPath := filepath.Join(dbDirPath, NewFragmentName())
	if err := ioutil.WriteFile(fragmentPath, sealed, 0644); err != nil {
		panic(err)
	}
}

// InsertByUpdatedAt 把 mima 插入到适当的位置, 使链表保持有序.
func (db *MimaItems) insertByUpdatedAt(mima *Mima) {
	if e := db.findUpdatedBefore(mima); e != nil {
		db.Items.InsertBefore(mima, e)
	} else {
		db.Items.PushBack(mima)
	}
}

// findUpdatedBefore 寻找一条记录, 其更新日期早于参数 mima 的更新日期.
// 如果找不到则返回 nil, 表示参数 mima 的更新日期是最早的.
func (db *MimaItems) findUpdatedBefore(mima *Mima) *list.Element {
	for e := db.Items.Front(); e != nil; e = e.Next() {
		v := e.Value.(*Mima)
		if v.UpdatedAt <= mima.UpdatedAt {
			return e
		}
	}
	return nil
}
