package main

import (
	"encoding/json"
	"log"
	"time"

	"golang.org/x/crypto/nacl/secretbox"
)

// Mima 用来表示一条记录.
// 其中, 标题是必须的, 别名是准唯一的, Nonce 是必须且唯一的.
type Mima struct {

	// (主键) (必须) (唯一) (自增)
	ID int

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
	// (注意 UpdatedAt 等于 原UpdatedAt 的情况, 可能表示软删除)
	// 当 UpdatedAt 为零, 表示需要彻底删除.
	UpdatedAt int64

	// 当 DeletedAt 大于零, 表示需要软删除.
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

// DecryptToMima 从已加密数据中解密出一个 Mima 来.
// 用于从数据库文件中读取数据进内存数据库.
func DecryptToMima(box []byte, key SecretKey) (*Mima, bool) {
	if len(box) < NonceSize {
		log.Println("It's not a secretbox.")
		return nil, false
	}
	var nonce Nonce
	copy(nonce[:], box[:NonceSize])
	mimaJSON, ok := secretbox.Open(nil, box[NonceSize:], &nonce, key)
	if !ok {
		return nil, false
	}
	var mima = new(Mima)
	if err := json.Unmarshal(mimaJSON, mima); err != nil {
		log.Println(err)
		return nil, false
	}
	return mima, true
}

// Update 以数据库碎片中的内容为准, 更新内存中的条目. (不包括软删除)
func (mima *Mima) Update(fragment *Mima) {
	mima.Title = fragment.Title
	mima.Alias = fragment.Alias
	mima.Username = fragment.Username
	mima.Password = fragment.Password
	mima.Notes = fragment.Notes
	mima.Favorite = fragment.Favorite
	mima.UpdatedAt = fragment.UpdatedAt
	mima.HistoryItems = fragment.HistoryItems
}

// Delete 更新删除时间, 即软删除.
func (mima *Mima) Delete() {
	mima.DeletedAt = time.Now().UnixNano()
}

// Seal 先把 mima 转换为 json, 再加密并返回二进制数据.
func (mima *Mima) Seal(key SecretKey) []byte {
	return secretbox.Seal(mima.Nonce[:], mima.toJSON(), &mima.Nonce, key)
}

// toJSON 把 mima 转换为 json 二进制数据.
func (mima *Mima) toJSON() []byte {
	blob, err := json.Marshal(mima)
	if err != nil {
		panic(err)
	}
	return blob
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
