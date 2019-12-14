package main

import (
	"container/list"
	"io/ioutil"
	"os"
	"time"
)

// MimaItems 相当于一个数据表.
type MimaItems struct {

	// 原始数据, 按 UpdatedAt 排序.
	Items *list.List
}

// NewMimaItems 生成一个新的 MimaItems, 并对其中的 Items 进行初始化.
func NewMimaItems() *MimaItems {
	items := new(MimaItems)
	items.Items = list.New()
	return items
}

// Init 初始化数据库.
// 即, 生成第一条记录, 并生成 mima.db
func (db *MimaItems) Init() {}

// MakeFirstMima 生成第一条记录, 用于保存密码.
func (db *MimaItems) MakeFirstMima(key SecretKey) {
	if _, err := os.Stat(dbFullPath); os.IsExist(err) {
		panic("数据库文件已存在, 不可重复创建")
	}
	mima := db.NewMima("")
	mima.Notes = randomString()
	sealed := mima.Seal(key)
	if err := ioutil.WriteFile(dbFullPath, sealed, 0644); err != nil {
		panic(err)
	}
}

// NewMima 生成一条新的记录, 插入到 MimaItems 里适当的位置, 并返回这条新记录.
func (db *MimaItems) NewMima(title string) *Mima {
	mima := new(Mima)

	if db.Items.Len() > 0 && len(title) == 0 {
		panic("Title 标题长度必须大于零")
	}

	mima.Nonce = newNonce()
	mima.CreatedAt = time.Now().Unix()
	mima.UpdatedAt = mima.CreatedAt

	db.InsertByUpdatedAt(mima)
	return mima
}

// InsertByUpdatedAt 把 mima 插入到适当的位置, 使链表保持有序.
func (db *MimaItems) InsertByUpdatedAt(mima *Mima) {
	if db.Items.Len() == 0 {
		db.Items.PushFront(mima)
		return
	}
	if e := db.findUpdatedBefore(mima); e != nil {
		db.Items.InsertBefore(mima, e)
	} else {
		db.Items.PushBack(mima)
	}
}

// findUpdatedBefore 寻找一条记录, 其更新日期早于参数 mima 的更新日期.
// 如果找不到则返回 nil, 参数 mima 的更新日期是最早的.
func (db *MimaItems) findUpdatedBefore(mima *Mima) *list.Element {
	for e := db.Items.Front(); e != nil; e = e.Next() {
		v := e.Value.(*Mima)
		if v.UpdatedAt <= mima.UpdatedAt {
			return e
		}
	}
	return nil
}
