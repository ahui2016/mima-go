package main

import (
	"bufio"
	"container/list"
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
	Undelete
	DeleteForever
)

// MimaDB 相当于一个数据表.
// 使用时注意需要利用 sync.RWMutex 上锁.
type MimaDB struct {
	sync.RWMutex

	// 主要用于 MimaDB.Add 中
	NextID int

	// 原始数据, 按 UpdatedAt 排序.
	Items *list.List

	key       SecretKey
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
		Items:     list.New(),
		key:       key,
		StartedAt: time.Now(),
		Period:    time.Minute * 5,
	}
}

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
		return tarballFile, dbFileNotFound
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

// ToSlice 把 list 转换为 slice, 保持其中元素的顺序.
func (db *MimaDB) ToSlice() (mimaSlice []*Mima) {
	for e := db.Items.Front(); e != nil; e = e.Next() {
		mima := e.Value.(*Mima)
		mimaSlice = append(mimaSlice, mima)
	}
	return
}

// All 返回全部 Mima, 但不包含 ID:0, 也不包含已软删除的条目.
// 并且, 不包含密码和备注. 另外, 更新时间最新(最近)的排在前面.
func (db *MimaDB) All() (all []*MimaForm) {
	if db.Items.Len() < 2 {
		return nil
	}
	for e := db.Items.Back(); e.Prev() != nil; e = e.Prev() {
		mima := e.Value.(*Mima)
		if mima.IsDeleted() {
			continue
		}
		form := mima.ToMimaForm().HidePasswordNotes()
		all = append(all, form)
	}
	return
}

// DeletedMimas 返回全部被软删除的 Mima, 不包含密码.
// 删除日期最新(最近)的排在前面.
func (db *MimaDB) DeletedMimas() (deleted []*MimaForm) {
	for e := db.Items.Back(); e.Prev() != nil; e = e.Prev() {
		mima := e.Value.(*Mima)
		if !mima.IsDeleted() {
			continue
		}
		form := mima.ToMimaForm().HidePasswordNotes()
		deleted = append(deleted, form)
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
	defer file.Close()

	for scanner.Scan() {
		box64 := scanner.Text()
		mima, err := DecryptToMima(box64, db.key)
		if err != nil {
			return fmt.Errorf("在初始化阶段解密失败: %w", err)
		}
		mima.Operation = 0
		mima.ID = db.NextID
		db.NextID++
		db.Items.PushBack(mima)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// readFragFilesAndUpdate 读取数据库碎片文件, 并根据其内容更新内存数据库.
// 分为 新增, 更新, 软删除, 彻底删除 四种情形.
func (db *MimaDB) readFragFilesAndUpdate(filePaths []string) error {
	for _, f := range filePaths {
		mima, err := readAndDecrypt(f, db.key)
		if err != nil {
			return err
		}
		if mima.Operation == Insert {
			db.insertByUpdatedAt(mima)
			continue
		}

		item, err := db.GetByID(mima.ID)
		if err != nil {
			return err
		}

		switch mima.Operation {
		case Insert: // 上面已操作, 这里不需要再操作.
		case Update:
			item.UpdateFromFrag(mima)
		case SoftDelete:
			item.Delete()
		case Undelete:
			item.Undelete()
		case DeleteForever:
			if _, err := db.deleteByID(item.ID); err != nil {
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
	defer dbFile.Close()

	dbWriter := bufio.NewWriter(dbFile)
	for e := db.Items.Front(); e != nil; e = e.Next() {
		mima := e.Value.(*Mima)
		box64, err := mima.Seal(db.key)
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
// 同时会生成数据库文件 mimadb/mima.db
func (db *MimaDB) MakeFirstMima() error {
	if !dbFileIsNotExist() {
		return errors.New("数据库文件已存在, 不可重复创建")
	}
	mima, err := NewMima("")
	if err != nil {
		return err
	}
	// mima.ID = 0 默认为零
	mima.Notes = randomString()
	db.Add(mima)
	box64, err := mima.Seal(db.key)
	if err != nil {
		return err
	}
	return writeFile(dbFullPath, box64)
}

// GetByID 凭 id 找 mima, 如果找不到就返回 nil. 忽略 id:0.
// 只有一种错误: 找不到记录.
func (db *MimaDB) GetByID(id int) (*Mima, error) {
	if e := db.getElementByID(id); e != nil {
		return e.Value.(*Mima), nil
	}
	return nil, fmt.Errorf("NotFound: 找不到 id: %d 的记录", id)
}

// GetFormByID 凭 id 找 mima 并转换为 MimaForm. 忽略 id:0.
// 只有一种错误: 找不到记录, 并且该错误信息已内嵌到 MimaForm 中.
func (db *MimaDB) GetFormByID(id int) *MimaForm {
	mima, err := db.GetByID(id)
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
	for e := db.Items.Back(); e.Prev() != nil; e = e.Prev() {
		mima := e.Value.(*Mima)
		if mima.IsDeleted() {
			continue
		}
		if mima.Alias == alias {
			return mima
		}
	}
	return nil
}

func (db *MimaDB) getElementByID(id int) *list.Element {
	// 这里的算法效率不高, 当预估数据量较大时需要改用更高效率的算法.
	for e := db.Items.Back(); e.Prev() != nil; e = e.Prev() { // 忽略 id:0
		mima := e.Value.(*Mima)
		if mima.ID == id {
			return e
		}
	}
	return nil
}

// Add 新增一个 mima 到数据库中, 并生成一块数据库碎片.
// 此时不检查 Alias 冲突, 因为此时不新增 Alias. 只能在 Edit 时增加新的 Alias.
func (db *MimaDB) Add(mima *Mima) error {
	if db.Items.Len() == 0 {
		// 第一条记录特殊处理,
		// 尤其注意从数据库文件读取数据到内存时, 确保第一条读入的是那条特殊记录.
		db.Items.PushFront(mima)
		db.NextID++
		return nil
	}
	mima.Title = strings.TrimSpace(mima.Title)
	if len(mima.Title) == 0 {
		return errNeedTitle
	}
	mima.ID = db.NextID
	db.NextID++
	db.insertByUpdatedAt(mima)

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
	mima, err := db.GetByID(form.ID)
	if err != nil {
		return err
	}
	mima.UpdateFromForm(form)
	return db.sealAndWriteFrag(mima, Update)
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
func (db *MimaDB) TrashByID(id int) error {
	mima, err := db.GetByID(id)
	if err != nil {
		return err
	}
	mima.Delete()
	return db.sealAndWriteFrag(mima, SoftDelete)
}

// UndeleteByID 从回收站中还原一个 mima (DeletedAt 重置为零), 并生成一块数据库碎片.
// 此时, 需要判断 Alias 有无冲突, 如有冲突则清空本条记录的 Alias.
func (db *MimaDB) UndeleteByID(id int) (err error) {
	mima, err := db.GetByID(id)
	if err != nil {
		return err
	}
	if db.IsAliasExist(mima.Alias) {
		err = fmt.Errorf("%w: %s, 因此此记录的 alias 已被清空", errAliasExist, mima.Alias)
		mima.Alias = ""
	}
	mima.Undelete()
	if err2 := db.sealAndWriteFrag(mima, Undelete); err2 != nil {
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

// deleteByID 删除内存数据库中的指定条目.
func (db *MimaDB) deleteByID(id int) (*Mima, error) {
	e := db.getElementByID(id)
	if e == nil {
		return nil, fmt.Errorf("NotFound: 找不到 id: %d 的条目", id)
	}
	value := db.Items.Remove(e)
	mima := value.(*Mima)
	return mima, nil
}

func (db *MimaDB) isEmpty() bool {
	if db.Items.Len() == 0 {
		return true
	}
	return false
}

// InsertByUpdatedAt 把 mima 插入到适当的位置, 使链表保持有序.
func (db *MimaDB) insertByUpdatedAt(mima *Mima) {
	if e := db.findUpdatedAfter(mima); e != nil {
		db.Items.InsertAfter(mima, e)
	} else {
		db.Items.PushBack(mima)
	}
}

// findUpdatedAfter 寻找一条记录, 其更新日期大于(晚于)参数 mima 的更新日期.
// 如果找不到则返回 nil, 表示参数 mima 的更新日期是最晚(最近)的.
func (db *MimaDB) findUpdatedAfter(mima *Mima) *list.Element {
	for e := db.Items.Front(); e != nil; e = e.Next() {
		v := e.Value.(*Mima)
		if v.UpdatedAt >= mima.UpdatedAt {
			return e
		}
	}
	return nil
}
