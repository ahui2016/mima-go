package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
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
	// 并且, 回收站里的 (DeletedAt 大于零的) 允许 Alias 重复.
	// 注意从回收站恢复条目时 (把 DeletedAt 重置为零时), 需要检查 Alias 冲突.
	Alias string

	// 一次性随机码, 用于加密 (必须) (唯一)
	// 但鉴于 Nonce 具有足够的长度使随机生成的 nonce 也不用担心重复,
	// 因此平时可偷懒不检查其唯一性.
	Nonce Nonce

	Username  string
	Password  string
	Notes     string
	CreatedAt int64
	UpdatedAt int64
	DeletedAt int64
	Operation Operation

	// 修改历史
	HistoryItems []History
}

// NewMima 生成一个新的 mima.
func NewMima(title string) (*Mima, error) {
	nonce, err := newNonce()
	if err != nil {
		return nil, err
	}
	mima := new(Mima)
	mima.Title = title
	mima.Nonce = nonce
	mima.CreatedAt = time.Now().UnixNano()
	mima.UpdatedAt = mima.CreatedAt
	return mima, nil
}

// NewMimaFromForm 根据 form 的信息生成一个新的 mima.
func NewMimaFromForm(form *MimaForm) (mima *Mima, err error) {
	if mima, err = NewMima(form.Title); err != nil {
		return
	}
	mima.Username = form.Username
	mima.Password = form.Password
	mima.Notes = form.Notes
	return
}

// DecryptToMima 从已加密数据中解密出一个 Mima 来.
// 用于从数据库文件中读取数据进内存数据库.
func DecryptToMima(box64 string, key SecretKey) (*Mima, error) {
	box, err := base64.StdEncoding.DecodeString(box64)
	if err != nil {
		return nil, err
	}
	if len(box) < NonceSize {
		return nil, errors.New("it's not a secretbox")
	}
	var nonce Nonce
	copy(nonce[:], box[:NonceSize])
	mimaJSON, ok := secretbox.Open(nil, box[NonceSize:], &nonce, key)
	if !ok {
		return nil, errors.New("secretbox open fail")
	}
	var mima = new(Mima)
	if err := json.Unmarshal(mimaJSON, mima); err != nil {
		return nil, err
	}
	return mima, nil
}

// UpdateFromFrag 以数据库碎片中的内容为准, 更新内存中的条目.
func (mima *Mima) UpdateFromFrag(fragment *Mima) {
	mima.Title = fragment.Title
	mima.Alias = fragment.Alias
	mima.Username = fragment.Username
	mima.Password = fragment.Password
	mima.Notes = fragment.Notes
	mima.UpdatedAt = fragment.UpdatedAt
	mima.HistoryItems = fragment.HistoryItems
}

// UpdateFromForm 以前端传回来的 MimaForm 为准, 更新内存中的条目内容.
func (mima *Mima) UpdateFromForm(form *MimaForm) {
	updatedAt := time.Now().UnixNano()
	mima.makeHistory(updatedAt)

	mima.Title = form.Title
	mima.Alias = form.Alias
	mima.Username = form.Username
	mima.Password = form.Password
	mima.Notes = form.Notes
	mima.UpdatedAt = updatedAt
}

func (mima *Mima) makeHistory(updatedAt int64) {
	h := History{
		Title:     mima.Title,
		Username:  mima.Username,
		Password:  mima.Password,
		Notes:     mima.Notes,
		UpdatedAt: updatedAt,
	}
	mima.HistoryItems = append(mima.HistoryItems, h)
}

// Delete 更新删除时间, 即软删除.
func (mima *Mima) Delete() {
	mima.DeletedAt = time.Now().UnixNano()
}

// IsDeleted 检查该 mima 是否已被软删除.
func (mima *Mima) IsDeleted() bool {
	return mima.DeletedAt > 0
}

// Undelete 把删除时间重置为零.
func (mima *Mima) Undelete() {
	mima.DeletedAt = 0
}

// Seal 先把 mima 转换为 json, 再加密并返回 base64 字符串.
func (mima *Mima) Seal(key SecretKey) (box64 string, err error) {
	var mimaJSON []byte
	mimaJSON, err = json.Marshal(mima)
	if err != nil {
		return
	}
	box := secretbox.Seal(mima.Nonce[:], mimaJSON, &mima.Nonce, key)
	box64 = base64.StdEncoding.EncodeToString(box)
	return
}

// ToMimaForm 把 Mima 转换为 MimaForm, 用于与前端网页交流.
func (mima *Mima) ToMimaForm() *MimaForm {
	var createdAt, updatedAt, deletedAt string
	if mima.CreatedAt > 0 {
		createdAt = time.Unix(0, mima.CreatedAt).Format(dateAndTime)
	}
	if mima.UpdatedAt > 0 {
		updatedAt = time.Unix(0, mima.UpdatedAt).Format(dateAndTime)
	}
	if mima.DeletedAt > 0 {
		deletedAt = time.Unix(0, mima.DeletedAt).Format(dateAndTime)
	}
	return &MimaForm{
		ID:        mima.ID,
		Title:     mima.Title,
		Alias:     mima.Alias,
		Username:  mima.Username,
		Password:  mima.Password,
		Notes:     mima.Notes,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		DeletedAt: deletedAt,
	}
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
