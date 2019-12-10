package main

import (
	"container/list"
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

// NewMima 生成一条新的记录, 插入到 MimaItems 里适当的位置, 并返回这条新记录.
func (table *MimaItems) NewMima(title string) *Mima {
	mima := new(Mima)

	// TODO: 在初始化时插入第一条记录, 然后这里可以简化.
	if table.Items.Len() > 0 && len(title) == 0 {
		panic("Title 标题长度必须大于零")
	}

	mima.Nonce = newNonce()
	mima.CreatedAt = time.Now().Unix()
	mima.UpdatedAt = mima.CreatedAt

	table.InsertByUpdatedAt(mima)
	return mima
}

// InsertByUpdatedAt 把 mima 插入到适当的位置, 使链表保持有序.
func (table *MimaItems) InsertByUpdatedAt(mima *Mima) {
	if table.Items.Len() == 0 {
		table.Items.PushFront(mima)
		return
	}
	if e := table.findUpdatedBefore(mima); e != nil {
		table.Items.InsertBefore(mima, e)
	} else {
		table.Items.PushBack(mima)
	}
}

// findUpdatedBefore 寻找一条记录, 其更新日期早于参数 mima 的更新日期.
// 如果找不到则返回 nil, 参数 mima 的更新日期是最早的.
func (table *MimaItems) findUpdatedBefore(mima *Mima) *list.Element {
	for e := table.Items.Front(); e != nil; e = e.Next() {
		v := e.Value.(*Mima)
		if v.UpdatedAt <= mima.UpdatedAt {
			return e
		}
	}
	return nil
}

// Mima 用来表示一条记录.
// 其中, 标题是必须的, 别名是准唯一的, Nonce 是必须且唯一的.
type Mima struct {

	// 标题 (必须)
	// 第一条记录的 Title 长度为零, 其他记录要求 Title 长度大于零.
	Title string

	// 别名, 用于辅助快速搜索 (准唯一)
	// 特别是用于命令行, 有一个快速定位功能.
	// 准唯一: 长度为零时允许重复, 长度大于零时要求唯一.
	Alias string

	// 一次性随机码, 用于加密 (必须) (唯一)
	// 但鉴于 Nonce 具有足够的长度使随机生成的 nonce 也不用担心重复,
	// 因此平时可偷懒不检查其唯一性.
	Nonce [NonceSize]byte

	// 用户名
	Username string

	// 密码
	Password string

	// 备注
	Notes string

	// 顶置
	Favorite bool

	// 创建时间
	CreatedAt int64

	// 更新时间
	UpdatedAt int64

	// 删除时间
	DeletedAt int64

	// 修改历史
	HistoryItems []History
}

// History 用来保存修改历史.
// 全部内容均直接保留当时的 Mima 内容, 不作任何修改.
type History struct {
	Title     string
	Username  string
	Password  string
	Notes     string
	UpdatedAt int64
}
