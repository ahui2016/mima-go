package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
	"time"

	"golang.org/x/crypto/nacl/secretbox"
)

// Mima 用来表示一条记录.
// 其中, 标题是必须的, 别名是准唯一的, Nonce 是必须且唯一的.
type Mima struct {

	// (主键) (必须) (唯一) (自增)
	ID string

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
	History []*History
}

// NewMima 生成一个新的 mima.
func NewMima(title string) (*Mima, error) {
	nonce, err := newNonce()
	if err != nil {
		return nil, err
	}
	mima := new(Mima)
	if mima.ID, err = newID(); err != nil {
		return nil, err
	}
	mima.Title = title
	mima.Nonce = nonce
	mima.CreatedAt = time.Now().UnixNano()
	mima.UpdatedAt = mima.CreatedAt
	return mima, nil
}

// newID 返回一个由时间戳和随机数组成的 id, 经测试瞬间生成一万个 id 不会重复.
// 由于时间戳的精度为秒, 因此如果两次生成 id 之间超过一秒, 则绝对不会重复.
func newID() (id string, err error) {
	var max int64 = 100_000_000
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return
	}
	timestamp := time.Now().Unix()
	idInt64 := timestamp*max + n.Int64()
	id = strconv.FormatInt(idInt64, 36)
	return
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
func (mima *Mima) UpdateFromFrag(fragment *Mima) (needChangeIndex bool) {
	// Alias 或 History 有可能发生了更改 (即使更新日期没有变化)
	mima.Alias = fragment.Alias
	mima.History = fragment.History

	if mima.UpdatedAt == fragment.UpdatedAt {
		return false
	}
	mima.Title = fragment.Title
	mima.Username = fragment.Username
	mima.Password = fragment.Password
	mima.Notes = fragment.Notes
	mima.UpdatedAt = fragment.UpdatedAt
	return true
}

// UpdateFromForm 以前端传回来的 MimaForm 为准, 更新内存中的条目内容.
// 如果只有 Alias 发生改变, 则改变 Alias, 但不生成历史记录, 也不移动元素.
func (mima *Mima) UpdateFromForm(form *MimaForm) (needChangeIndex bool, needWriteFrag bool, err error) {
	if mima.Alias != form.Alias {
		mima.Alias = form.Alias
		needWriteFrag = true
	}
	if mima.equalToForm(form) {
		return false, needWriteFrag, nil
	}
	updatedAt := time.Now().UnixNano()
	if err = mima.makeHistory(updatedAt); err != nil {
		return
	}

	mima.Title = form.Title
	mima.Username = form.Username
	mima.Password = form.Password
	mima.Notes = form.Notes
	mima.UpdatedAt = updatedAt
	return true, true, nil
}

// equalToForm 用于检查 mima 与 form 的内容是否需要基本相等.
// 如果基本相等则返回 true.
// 注意本函数不检查 Alias.
func (mima *Mima) equalToForm(form *MimaForm) bool {
	s1 := mima.Title + mima.Username + mima.Password + mima.Notes
	s2 := form.Title + form.Username + form.Password + form.Notes
	return s1 == s2
}

func (mima *Mima) makeHistory(updatedAt int64) error {
	datetime := time.Unix(0, updatedAt).Format(dateAndTime)
	for _, v := range mima.History {
		if v.DateTime == datetime {
			return errors.New("历史记录的 DateTime 发生重复")
		}
	}
	h := &History{
		Title:    mima.Title,
		Username: mima.Username,
		Password: mima.Password,
		Notes:    mima.Notes,
		DateTime: datetime,
	}
	mima.History = append([]*History{h}, mima.History...)
	return nil
}

// Delete 更新删除时间, 即软删除.
func (mima *Mima) Delete() {
	mima.DeletedAt = time.Now().UnixNano()
}

// IsDeleted 检查该 mima 是否已被软删除.
func (mima *Mima) IsDeleted() bool {
	return mima.DeletedAt > 0
}

// UnDelete 把删除时间重置为零.
func (mima *Mima) UnDelete() {
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
	form := mima.ToFormWithHistory()
	form.History = nil
	return form
}

// ToFormWithHistory 把 Mima 转换为有 History 的 MimaForm, 主要用于 edit 页面.
func (mima *Mima) ToFormWithHistory() *MimaForm {
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
		History:   mima.History,
	}
}

// DeleteHistory 删除一条历史记录.
// 这里不检查第 i 条历史记录是否存在, 请使用 GetHistory 获得正确的 i.
func (mima *Mima) DeleteHistory(i int) {
	mima.History = append(mima.History[:i], mima.History[i+1:]...)
}

func (mima *Mima) GetHistory(datetime string) (i int, item *History, ok bool) {
	ok = true
	for i, item = range mima.History {
		if item.DateTime == datetime {
			return
		}
	}
	return i, item, false
}

// History 用来保存修改历史.
// 全部内容均直接保留当时的 Mima 内容, 不作任何修改.
type History struct {
	Title    string
	Username string
	Password string
	Notes    string

	// 考虑到实际使用情景, 在一个 mima 的历史记录里面,
	// DateTime 应该是唯一的 (同一条记录不可能同时修改两次).
	DateTime string
	ToDelete bool
}
