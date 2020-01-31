package db

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"golang.org/x/crypto/nacl/secretbox"
	"time"
)

// Mima 用来表示一条记录.
// 其中, 标题是必须的, Nonce 是必须且唯一的.
type Mima struct {

	// (主键) (必须) (唯一)
	// 由于根据实际使用的性能要求, 采用足以避免重复的算法, 因此平时可偷懒不检查其唯一性.
	ID string

	// 标题 (必须)
	// 第一条记录的 Title 长度为零, 其他记录要求 Title 长度大于零.
	Title string

	// 别名, 用于辅助快速搜索 (允许重复)
	// 多个条目共用同一个别名, 可模拟一个条目拥有多个密码的情形.
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

// Decrypt 从已加密数据中解密出一个 Mima 来.
// 用于从数据库文件中读取数据进内存数据库 (DB.readFullPath).
func Decrypt(box64 string, key *SecretKey) (*Mima, error) {
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
		return nil, errors.New("Mima.Decrypt: secretbox open fail")
	}
	mima := new(Mima)
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

// Delete 更新删除时间, 即软删除.
func (mima *Mima) Delete() {
	mima.DeletedAt = time.Now().UnixNano()
}

// UnDelete 把删除时间重置为零.
func (mima *Mima) UnDelete() {
	mima.DeletedAt = 0
}

// ToFormWithHistory 把 Mima 转换为有 History 的 MimaForm, 主要用于 edit 页面.
func (mima *Mima) ToForm() *MimaForm {
	var createdAt, updatedAt, deletedAt string
	if mima.CreatedAt > 0 {
		createdAt = time.Unix(0, mima.CreatedAt).Format(DateTimeFormat)
	}
	if mima.UpdatedAt > 0 {
		updatedAt = time.Unix(0, mima.UpdatedAt).Format(DateTimeFormat)
	}
	if mima.DeletedAt > 0 {
		deletedAt = time.Unix(0, mima.DeletedAt).Format(DateTimeFormat)
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
	datetime := time.Unix(0, updatedAt).Format(DateTimeFormat)
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

// Seal 先把 mima 转换为 json, 再加密并返回 base64 字节码.
func (mima *Mima) Seal(key *SecretKey) (string, error) {
	mimaJSON, err := json.Marshal(mima)
	if err != nil {
		return "", err
	}
	box := secretbox.Seal(mima.Nonce[:], mimaJSON, &mima.Nonce, key)
	return base64.StdEncoding.EncodeToString(box), nil
}

// DeleteHistory 彻底删除一条历史记录.
func (mima *Mima) DeleteHistory(datetime string) error {
	if i := mima.getHistory(datetime); i < 0 {
		return errors.New("找不到历史记录:" + datetime)
	} else {
		mima.History = append(mima.History[:i], mima.History[i+1:]...)
		return nil
	}
}

func (mima *Mima) getHistory(datetime string) int {
	for i, item := range mima.History {
		if item.DateTime == datetime {
			return i
		}
	}
	return -1
}

func (mima *Mima) IsDeleted() bool {
	return mima.DeletedAt > 0
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
}

// MimaForm 用前端显示一个 Mima.
type MimaForm struct {
	ID        string
	Title     string
	Alias     string
	Username  string
	Password  string
	Notes     string
	CreatedAt string
	UpdatedAt string
	DeletedAt string
	History   []*History
	Err       error
	Info      error
}

// HideSecret 删除密码, 备注以及历史记录等敏感信息, 用于不需要展示密码的页面 (为了提高安全性).
func (form *MimaForm) HideSecrets() *MimaForm {
	if len(form.Password) > 0 {
		form.Password = "******"
	}
	form.Notes = ""
	form.History = nil
	return form
}

// IsDeleted 检查该 form 所对应的 mima 是否已被软删除.
func (form *MimaForm) IsDeleted() bool {
	return form.DeletedAt != ""
}

type SearchResult struct {
	SearchText string
	Forms      []*MimaForm
	Info       error
	Err        error
}

type AjaxResponse struct {
	Message string
}

// Feedback 用来表示一个普通的表单.
type Feedback struct {
	Msg  string
	Err  error
	Info error
}
