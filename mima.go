package main

import (
	"encoding/json"
	"time"

	"golang.org/x/crypto/nacl/secretbox"
)

// Mima 用来表示一条记录.
// 其中, 标题是必须的, 别名是准唯一的, Nonce 是必须且唯一的.
type Mima struct {

	// (主键) (必须) (唯一) (自增)
	ID uint

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
	Nonce Nonce

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

	// 当 UpdatedAt 等于 CreatedAt, 表示新增.
	// 当 UpdatedAt 大于 CreatedAt, 表示更新.
	// 当 UpdatedAt 为零, 表示需要彻底删除.
	UpdatedAt int64

	// 删除时间
	DeletedAt int64

	// 修改历史
	HistoryItems []History
}

// NewMima 生成一个新的 mima.
func NewMima(title string) *Mima {
	mima := new(Mima)
	mima.Title = title
	mima.Nonce = newNonce()
	mima.CreatedAt = time.Now().UnixNano()
	mima.UpdatedAt = mima.CreatedAt
	return mima
}

// ToJSON 把 mima 转换为 json 二进制数据.
func (mima *Mima) ToJSON() []byte {
	blob, err := json.Marshal(mima)
	if err != nil {
		panic(err)
	}
	return blob
}

// Seal 先把 mima 转换为 json, 再加密并返回二进制数据.
func (mima *Mima) Seal(key SecretKey) []byte {
	return secretbox.Seal(mima.Nonce[:], mima.ToJSON(), &mima.Nonce, key)
}

// History 用来保存修改历史.
// 全部内容均直接保留当时的 Mima 内容, 不作任何修改.
type History struct {
	Title    string
	Username string
	Password string
	Notes    string

	// 考虑到实际使用情景, 在一个 mima 的历史记录里面,
	// UpdatedAt 应该是唯一的 (同一条记录不可能同时修改两次).
	UpdatedAt int64
}
