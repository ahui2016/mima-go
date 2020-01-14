package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ahui2016/mima-go/util"
)

// Operation 表示数据库的操作指令.
// 由于本程序不使用真正的数据库, 而是自己弄一个简陋的数据库, 因此需要该类型辅助.
type Operation int

// 数据库操作的 enum (枚举)
const (
	Insert Operation = iota + 1
	Update
	SoftDelete
	UnDelete
	DeleteForever
)

// MimaDB 相当于一个数据表.
// 使用时注意需要利用 sync.RWMutex 上锁.
type MimaDB struct {
	sync.RWMutex

	// 原始数据, 按 UpdatedAt 排序, 最新(最近)的在后面.
	Items []*Mima

	key       SecretKey
	userKey   SecretKey
	StartedAt time.Time
	Period    time.Duration
}

// NewMimaDB 生成一个新的 MimaDB, 并对其中的 Items 进行初始化.
func NewMimaDB(key SecretKey) *MimaDB {
	if key == nil {
		// 因为改错误属于 "编程时" 错误, 不是 "运行时" 错误, 可在运行前处理,
		// 因此不返回错误信息, 而是让程序直接崩溃.
		panic("缺少key, 需要key")
	}
	return &MimaDB{
		Items:     []*Mima{},
		userKey:   key,
		StartedAt: time.Now(),
		Period:    time.Minute * 5,
	}
}

// Len 返回 MimaDB.Items 的长度.
func (db *MimaDB) Len() int {
	return len(db.Items)
}

//func (db *MimaDB) lastIndex() int {
//	return db.Len() - 1
//}

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
		return "", dbFileNotFound
	}
	if err = db.scanDBtoMemory(); err != nil {
		return
	}
	if fragFiles, err = fragFilePaths(); err != nil {
		return
	}
	if len(fragFiles) == 0 {
		// 如果没有数据库碎片文件, Rebuild 就相当于只执行 scanDBtoMemory.
		return
	}
	if tarballFile, err = backupToTar(filesToBackup(fragFiles)); err != nil {
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

// All 返回全部 Mima, 但不包含 ID:0, 也不包含已软删除的条目.
// 并且, 不包含密码和备注. 另外, 更新时间最新(最近)的排在前面.
func (db *MimaDB) All() (all []*MimaForm) {
	if db.Len()-1 == 0 {
		return nil
	}
	for i := db.Len() - 1; i > 0; i-- {
		mima := db.Items[i].ToMimaForm().HidePasswordNotes()
		if mima.IsDeleted() {
			continue
		}
		all = append(all, mima)
	}
	return
}

// DeletedMimas 返回全部被软删除的 Mima, 不包含密码.
// 删除日期最新(最近)的排在前面.
func (db *MimaDB) DeletedMimas() (deleted []*MimaForm) {
	if db.Len()-1 == 0 {
		return nil
	}
	for i := db.Len() - 1; i > 0; i-- {
		mima := db.Items[i].ToMimaForm().HidePasswordNotes()
		if !mima.IsDeleted() {
			continue
		}
		deleted = append(deleted, mima)
	}
	sort.Slice(deleted, func(i, j int) bool {
		return deleted[i].DeletedAt > deleted[j].DeletedAt
	})
	return
}

func (db *MimaDB) deleteFragFiles(filePaths []string) error {
	for _, f := range filePaths {
		if err := os.Remove(f); err != nil {
			return err
		}
	}
	return nil
}

// scanDBtoMemory 读取 dbFullPath, 填充 MimaDB.
func (db *MimaDB) scanDBtoMemory() error {
	scanner, file, err := util.NewFileScanner(dbFullPath)
	if err != nil {
		return err
	}
	//noinspection GoUnhandledErrorResult
	defer file.Close()

	for scanner.Scan() {
		var (
			mima *Mima
			err  error
			key []byte
			box64 = scanner.Text()
		)
		if db.key == nil {
			if mima, err = DecryptToMima(box64, db.userKey); err != nil {
				return fmt.Errorf("用户密码错误: %w", err)
			}
			if key, err = base64.StdEncoding.DecodeString(mima.Password); err != nil {
				return err
			}
			db.SetKeyFromSlice(key)
		} else {
			if mima, err = DecryptToMima(box64, db.key); err != nil {
				return fmt.Errorf("用户密码正确, 但内部密码错误: %w", err)
			}
		}
		db.Items = append(db.Items, mima)
	}
	return scanner.Err()
}

func (db *MimaDB) SetKeyFromSlice(slice []byte) {
	var masterKey [KeySize]byte
	copy(masterKey[:], slice)
	db.key = &masterKey
}

// readFragFilesAndUpdate 读取数据库碎片文件, 并根据其内容更新内存数据库.
// 分为 新增, 更新, 软删除, 彻底删除 四种情形.
func (db *MimaDB) readFragFilesAndUpdate(filePaths []string) error {
	if !sort.StringsAreSorted(filePaths) {
		return errors.New("filePaths 必须从小到大排序")
	}
	for _, f := range filePaths {
		mima, err := readAndDecrypt(f, db.key)
		if err != nil {
			return err
		}

		if mima.Operation == Insert {
			db.Items = append(db.Items, mima)
			continue
		}

		i, item, err := db.GetByID(mima.ID)
		if err != nil {
			return err
		}

		switch mima.Operation {
		case Insert: // 上面已操作, 这里不需要再操作.
		case Update:
			item.UpdateFromFrag(mima)
			db.Items = append(db.Items, item)
			db.Items = append(db.Items[:i], db.Items[i+1:]...)
		case SoftDelete:
			item.Delete()
		case UnDelete:
			item.UnDelete()
		case DeleteForever:
			if err := db.deleteByID(item.ID); err != nil {
				return err
			}
		default: // 一共 5 种 Operation 已在上面全部处理, 没有其他可能.
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
	//noinspection GoUnhandledErrorResult
	defer dbFile.Close()

	dbWriter := bufio.NewWriter(dbFile)
	for i := 0; i < db.Len(); i++ {
		var key SecretKey
		if i == 0 {
			key = db.userKey
		} else {
			key = db.key
		}
		mima := db.Items[i]
		box64, err := mima.Seal(key)
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
// 第一条记录的 ID 特殊处理, 手动设置为空字符串.
// 同时会生成数据库文件 mimadb/mima.mdb
func (db *MimaDB) MakeFirstMima() error {
	if !dbFileIsNotExist() {
		return errors.New("数据库文件已存在, 不可重复创建")
	}
	mima, err := NewMima("")
	if err != nil {
		return err
	}
	key := db.newMasterKey()
	db.SetKeyFromSlice(key)
	mima.ID = ""
	mima.Password = base64.StdEncoding.EncodeToString(key)
	mima.Notes = randomString()
	db.Items = []*Mima{mima}
	box64, err := mima.Seal(db.userKey) // 第一条记录特殊处理, 用 userKey 加密.
	if err != nil {
		return err
	}
	return writeFile(dbFullPath, box64)
}

// newMasterKey 生成 master key, 由于需要保存在 Mima.Password 里, 因此采用 base64.
func (db *MimaDB) newMasterKey() []byte {
	password := randomString()
	key := sha256.Sum256([]byte(password))
	return key[:]
}

// GetByID 凭 id 找 mima. 忽略 id:0. 只有一种错误: 找不到记录.
func (db *MimaDB) GetByID(id string) (index int, mima *Mima, err error) {
	for index = 1; index < db.Len(); index++ {
		mima = db.Items[index]
		if mima.ID == id {
			return
		}
	}
	err = fmt.Errorf("NotFound: 找不到 id: %s 的记录", id)
	return
}

// GetFormByID 凭 id 找 mima 并转换为 MimaForm. 忽略 id:0.
// 只有一种错误: 找不到记录, 并且该错误信息已内嵌到 MimaForm 中.
func (db *MimaDB) GetFormByID(id string) *MimaForm {
	_, mima, err := db.GetByID(id)
	if err != nil {
		return &MimaForm{Err: err}
	}
	return mima.ToMimaForm()
}

// GetByAlias 凭 alias 找 mima, 如果找不到就返回 nil.
func (db *MimaDB) GetByAlias(alias string) *Mima {
	if alias == "" {
		return nil
	}
	for i := 1; i < db.Len(); i++ {
		mima := db.Items[i]
		if mima.IsDeleted() {
			continue
		}
		if mima.Alias == alias {
			return mima
		}
	}
	return nil
}

// Add 新增一个 mima 到数据库中, 并生成一块数据库碎片.
// 此时不检查 Alias 冲突, 因为此时不新增 Alias. 只能在 Edit 时增加新的 Alias.
// 此时不重新排序, 新 mima 直接加到最后, 因为新记录的更新日期必然是最新的.
func (db *MimaDB) Add(mima *Mima) error {
	mima.Title = strings.TrimSpace(mima.Title)
	if len(mima.Title) == 0 {
		return errNeedTitle
	}
	db.Items = append(db.Items, mima)
	return db.sealAndWriteFrag(mima, Insert)
}

// Update 根据 MimaForm 更新对应的 Mima 内容, 并生成一块数据库碎片.
func (db *MimaDB) Update(form *MimaForm) error {
	if len(form.Title) == 0 {
		return errNeedTitle
	}
	if db.IsAliasExist(form.Alias) {
		return errAliasExist
	}
	i, mima, err := db.GetByID(form.ID)
	if err != nil {
		return err
	}
	if mima.UpdateFromForm(form) {
		// 如果实际上未发生更新, 则不需要移动元素.
		db.Items = append(db.Items, mima)
		db.Items = append(db.Items[:i], db.Items[i+1:]...)
	}
	return mdb.sealAndWriteFrag(mima, Update)
}

func (db *MimaDB) sealAndWriteFrag(mima *Mima, op Operation) error {
	mima.Operation = op
	sealed, err := mima.Seal(db.key)
	if err != nil {
		return err
	}
	return writeFragFile(sealed)
}

// TrashByID 软删除一个 mima, 并生成一块数据库碎片.
func (db *MimaDB) TrashByID(id string) error {
	_, mima, err := db.GetByID(id)
	if err != nil {
		return err
	}
	mima.Delete()
	return db.sealAndWriteFrag(mima, SoftDelete)
}

// UnDeleteByID 从回收站中还原一个 mima (DeletedAt 重置为零), 并生成一块数据库碎片.
// 此时, 需要判断 Alias 有无冲突, 如有冲突则清空本条记录的 Alias.
func (db *MimaDB) UnDeleteByID(id string) (err error) {
	_, mima, err := db.GetByID(id)
	if err != nil {
		return err
	}
	if db.IsAliasExist(mima.Alias) {
		err = fmt.Errorf("%w: %s, 因此该记录的 alias 已被清空", errAliasExist, mima.Alias)
		mima.Alias = ""
	}
	mima.UnDelete()
	if err2 := db.sealAndWriteFrag(mima, UnDelete); err2 != nil {
		return err2
	}
	return
}

// IsAliasExist 判断 alias 有无冲突.
func (db *MimaDB) IsAliasExist(alias string) (ok bool) {
	if m := db.GetByAlias(alias); m != nil {
		ok = true
	}
	return
}

// deleteByID 删除内存数据库中的指定记录, 不生成数据库碎片.
// 用于 ReBuild 时根据数据库碎片删除记录.
func (db *MimaDB) deleteByID(id string) error {
	i, _, err := db.GetByID(id)
	if err != nil {
		return err
	}
	db.Items = append(db.Items[:i], db.Items[i+1:]...)
	return nil
}

func (db *MimaDB) isEmpty() bool {
	return db.Len() == 0
}

// insertByUpdatedAt 把 mima 插入到适当的位置, 使链表保持有序.
// 本函数假设 db.Items 已按更新日期从小到大排序, 先找到最大的更新日期, 把 mima 插入其前面.
/*
func (db *MimaDB) insertByUpdatedAt(mima *Mima) {
	switch i := db.findUpdatedAfter(mima); {
	case i == 0:
		panic("这里 index 不能为零, 因为 id:0 的记录应被避开")
	case i > 0:
		db.Items = append(append(db.Items[:i], mima), db.Items[i:]...)
	case i < 0:
		db.Items = append(db.Items, mima)
	}
}
*/

// findUpdatedAfter 寻找一条记录, 其更新日期大于(晚于)参数 mima 的更新日期.
// 本函数假设 db.Items 已按更新日期从小到大排序.
// 如果找不到则返回 -1, 表示参数 mima 的更新日期是最大(最新)的.
/*
func (db *MimaDB) findUpdatedAfter(mima *Mima) int {
	for i := 1; i < db.Len(); i++ { // 这里 i != 0, 避开 id:0 的记录.
		if db.Items[i].UpdatedAt >= mima.UpdatedAt {
			return i
		}
	}
	return -1
}
*/
