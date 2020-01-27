// 一个带有加密功能的简陋数据库.
package db

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/ahui2016/mima-go/tarball"
	"github.com/ahui2016/mima-go/util"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
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

// DB 相当于一个数据库.
// 其中 mimaTable 相当于一个数据表, Mima 相当于这个数据表的 schema.
type DB struct {
	// 每次使用 DB 时注意需要上锁.
	sync.RWMutex

	// 原始数据, 按 UpdatedAt 排序, 最新(最近)的在后面.
	mimaTable []*Mima

	// 由用户密码生成 userKey, 用来加密解密 key, 再用 key 去实际加密数据.
	userKey *SecretKey
	key     *SecretKey

	// 本数据库具有定时关闭功能, 这是数据库启动时刻和有效时长.
	startedAt time.Time
	validTerm time.Duration

	// 数据库文件的绝对路径, 备份文件夹的绝对路径.
	// 另外, 数据库碎片文件的后缀名和数据库备份文件的后缀名在 db/init.go 中定义.
	// 为了方便测试, 权限设为 public.
	FullPath  string
	BackupDir string
}

// NewDB 生成一个新的 DB. 此时, 内存数据库里没有数据, 也没有 key.
// 要么通过 DB.Init 生成新的数据库, 要么通过 DB.Rebuild 从文件中恢复数据库.
func NewDB(fullPath, backupDir string) *DB {
	return &DB{
		FullPath:  fullPath,
		BackupDir: backupDir,
	}
}

func (db *DB) Reset() {
	db.userKey = nil
	db.key = nil
	db.mimaTable = nil
}

func (db *DB) IsNotInit() bool {
	return db.userKey == nil || db.key == nil || db.Len() < 1
}

// Init 生成第一条记录, 用于保存密码.
// 第一条记录的 ID 特殊处理, 手动设置为空字符串.
// 同时会生成数据库文件 DB.FullPath
func (db *DB) Init(userKey *SecretKey) error {
	if !db.fileNotExist() {
		return errors.New("数据库文件已存在, 不可重复创建")
	}
	key := newRandomKey()
	db.key = &key
	db.userKey = userKey
	db.startedAt = time.Now()
	db.validTerm =  time.Minute * 5
	mima, err := NewMima("")
	if err != nil {
		return err
	}
	mima.ID = ""
	mima.Password = base64.StdEncoding.EncodeToString(key[:])
	mima.Notes = randomString()
	db.mimaTable = []*Mima{mima}
	box64, err := mima.Seal(db.userKey) // 第一条记录特殊处理, 用 userKey 加密.
	if err != nil {
		return err
	}
	return writeFile(db.FullPath, box64)
}

// Rebuild 填充内存数据库，读取数据库碎片, 整合到数据库文件中.
// 每次启动程序, 初始化时, 如果已有账号, 自动执行一次 Rebuild.
// 为了方便测试返回 tarball 文件路径.
func (db *DB) Rebuild(userKey *SecretKey) (tarballFile string, err error) {
	db.userKey = userKey
	if !db.isEmpty() {
		return tarballFile, errors.New("初始化失败: 内存中的数据库已有数据")
	}
	if db.fileNotExist() {
		return "", FileNotFound
	}
	if err = db.readFullPath(); err != nil {
		return
	}
	fragFiles, err := db.getFragPaths()
	if err != nil {
		return
	}
	if len(fragFiles) == 0 {
		// 如果没有数据库碎片文件, Rebuild 就相当于只执行 scanDBtoMemory.
		return
	}
	if tarballFile, err = db.backupToTar(db.filesToBackup(fragFiles)); err != nil {
		return
	}
	if err = db.readFragFilesAndUpdate(fragFiles); err != nil {
		return
	}
	if err = db.rewriteDBFile(); err != nil {
		return
	}
	err = deleteFiles(fragFiles)
	return
}

// rewriteDBFile 覆盖重写数据库文件, 将其更新为当前内存数据库的内容.
func (db *DB) rewriteDBFile() error {
	dbFile, err := os.Create(db.FullPath)
	if err != nil {
		return err
	}
	//noinspection GoUnhandledErrorResult
	defer dbFile.Close()

	dbWriter := bufio.NewWriter(dbFile)
	for i := 0; i < db.Len(); i++ {
		var key *SecretKey
		if i == 0 {
			key = db.userKey
		} else {
			key = db.key
		}
		mima := db.mimaTable[i]
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

// readFragFilesAndUpdate 读取数据库碎片文件, 并根据其内容更新内存数据库.
// 分为 新增, 更新, 软删除, 彻底删除 四种情形.
func (db *DB) readFragFilesAndUpdate(filePaths []string) error {
	if !sort.StringsAreSorted(filePaths) {
		return errors.New("filePaths 必须从小到大排序")
	}
	for _, f := range filePaths {
		frag, err := readAndDecrypt(f, db.key)
		if err != nil {
			return err
		}

		if frag.Operation == Insert {
			db.mimaTable = append(db.mimaTable, frag)
			continue
		}

		i, mima, err := db.GetByID(frag.ID)
		if err != nil {
			return err
		}

		switch frag.Operation {
		case Insert: // 上面已操作, 这里不需要再操作.
		case Update:
			if mima.UpdateFromFrag(frag) {
				db.mimaTable = append(db.mimaTable, mima)
				db.mimaTable = append(db.mimaTable[:i], db.mimaTable[i+1:]...)
			}
		case SoftDelete:
			mima.Delete()
		case UnDelete:
			mima.Alias = frag.Alias // 从垃圾桶里恢复时, Alias 有可能被删除.
			mima.UnDelete()
		case DeleteForever:
			if _, err := db.deleteByID(mima.ID); err != nil {
				return err
			}
		default: // 一共 5 种 Operation 已在上面全部处理, 没有其他可能.
		}
	}
	return nil
}

// deleteByID 删除内存数据库中的指定记录, 不生成数据库碎片.
// 用于 ReBuild 时根据数据库碎片删除记录.
func (db *DB) deleteByID(id string) (*Mima, error) {
	i, mima, err := db.GetByID(id)
	if err != nil {
		return nil, err
	}
	db.mimaTable = append(db.mimaTable[:i], db.mimaTable[i+1:]...)
	return mima, nil
}

// GetByIndex 为了测试方便.
func (db *DB) GetByIndex(i int) *Mima {
	return db.mimaTable[i]
}

// GetByID 凭 id 找 mima. 忽略 index:0. 只有一种错误: 找不到记录.
// 为什么找不到时要返回错误不返回 nil? 因为后续需要返回错误, 在这里集中处理更方便.
func (db *DB) GetByID(id string) (index int, mima *Mima, err error) {
	for index = 1; index < db.Len(); index++ {
		mima = db.mimaTable[index]
		if mima.ID == id {
			return
		}
	}
	err = fmt.Errorf("NotFound: 找不到 id: %s 的记录", id)
	return
}

// GetFormByID 凭 id 找 mima 并转换为有 History 的 MimaForm.
func (db *DB) GetFormByID(id string) *MimaForm {
	_, mima, err := db.GetByID(id)
	if err != nil {
		return &MimaForm{Err: err}
	}
	return mima.ToForm()
}

// GetByAlias 凭 alias 找 mima, 如果找不到就返回 nil.
func (db *DB) GetByAlias(alias string) *Mima {
	if alias == "" {
		return nil
	}
	for i := 1; i < db.Len(); i++ {
		mima := db.mimaTable[i]
		if mima.IsDeleted() {
			continue
		}
		if mima.Alias == alias {
			return mima
		}
	}
	return nil
}

// GetFormByAlias 凭 alias 找 mima, 并转换为 MimaForm 返回.
// 只有一种错误: 找不到记录, 并且该错误信息已内嵌到 MimaForm 中.
func (db *DB) GetFormByAlias(alias string) *MimaForm {
	mima := db.GetByAlias(alias)
	if mima == nil {
		err := fmt.Errorf("NotFound: 找不到 alias: %s 的记录", alias)
		return &MimaForm{Err: err}
	}
	return mima.ToForm().HideSecrets()
}
// backupToTar 把数据库文件以及碎片文件备份到一个 tarball 里.
// 主要在 Rebuild 之前使用, 以防万一 rebuild 出错.
// 为了方便测试返回 tarball 的完整路径.
func (db *DB) backupToTar(files []string) (filePath string, err error) {
	filePath = filepath.Join(db.BackupDir, newTimestampFilename(TarballExt))
	err = tarball.Create(filePath, files)
	return
}

// filesToBackup 返回需要备份的文件的完整路径.
func (db *DB) filesToBackup(fragFiles []string) []string {
	return append(fragFiles, db.FullPath)
}

// getFragPaths 返回数据库碎片文件的完整路径, 并且已排序.
// 必须确保从小到大排序.
func (db *DB) getFragPaths() ([]string, error) {
	pattern := filepath.Join(db.BackupDir, "*"+FragExt)
	filePaths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(filePaths)
	return filePaths, nil
}

func (db *DB) isEmpty() bool {
	return db.Len() == 0
}

func (db *DB) Len() int {
	return len(db.mimaTable)
}

func (db *DB) fileNotExist() bool {
	_, err := os.Stat(db.FullPath)
	if os.IsNotExist(err) {
		return true
	}
	if err != nil {
		panic(err)
	}
	return false
}

// readFullPath 读取 db.FullPath, 填充 db.
// blankMima 是一个空的 Mima 实体, 它携带着 Mima.Decrypt 的具体实现.
func (db *DB) readFullPath() (err error) {
	scanner, file, err := util.NewFileScanner(db.FullPath)
	if err != nil {
		return err
	}
	//noinspection GoUnhandledErrorResult
	defer file.Close()

	for scanner.Scan() {
		var mima *Mima
		box64 := scanner.Text()
		if db.key == nil {
			if mima, err = Decrypt(box64, db.userKey); err != nil {
				return fmt.Errorf("用户密码错误: %w", err)
			}
			keyBytes, err := base64.StdEncoding.DecodeString(mima.Password)
			if err != nil {
				return err
			}
			key := bytesToKey(keyBytes)
			db.key = &key
		} else {
			if mima, err = Decrypt(box64, db.key); err != nil {
				return fmt.Errorf("用户密码正确, 但内部密码错误: %w", err)
			}
		}
		db.mimaTable = append(db.mimaTable, mima)
	}
	return scanner.Err()
}

// MimaTable 为了测试方便.
func (db *DB) MimaTable() []*Mima {
	return db.mimaTable
}

// All 返回全部 Mima, 但不包含 index:0, 也不包含已软删除的条目.
// 并且删除含密码和备注等敏感信息. 另外, 更新时间最新(最近)的排在前面.
func (db *DB) All() (all []*MimaForm) {
	if db.Len()-1 == 0 {
		return
	}
	for i := db.Len() - 1; i > 0; i-- {
		mima := db.mimaTable[i].ToForm().HideSecrets()
		if mima.IsDeleted() {
			continue
		}
		all = append(all, mima)
	}
	return
}

// DeletedMimas 返回全部被软删除的 Mima, 不包含密码.
// 删除日期最新(最近)的排在前面.
func (db *DB) DeletedMimas() (deleted []*MimaForm) {
	if db.Len()-1 == 0 {
		return nil
	}
	for i := db.Len() - 1; i > 0; i-- {
		mima := db.mimaTable[i].ToForm().HideSecrets()
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

// Add 新增一个 mima 到数据库中, 并生成一块数据库碎片.
// 此时不检查 Alias 冲突, 因为此时不新增 Alias. 只能在 Edit 时增加新的 Alias.
// 此时不重新排序, 新 mima 直接加到最后, 因为新记录的更新日期必然是最新的.
func (db *DB) Add(mima *Mima) error {
	mima.Title = strings.TrimSpace(mima.Title)
	if len(mima.Title) == 0 {
		return errNeedTitle
	}
	db.mimaTable = append(db.mimaTable, mima)
	return db.sealAndWriteFrag(mima, Insert)
}

// Update 根据 MimaForm 更新对应的 Mima 内容, 并生成一块数据库碎片.
func (db *DB) Update(form *MimaForm) (err error) {
	if len(form.Title) == 0 {
		return errNeedTitle
	}
	if db.IsAliasConflicts(form.Alias, form.ID) {
		return ErrAliasConflicts
	}
	i, mima, err := db.GetByID(form.ID)
	if err != nil {
		return err
	}
	needChangeIndex, needWriteFrag, err := mima.UpdateFromForm(form)
	if err != nil {
		return err
	}
	if needChangeIndex {
		db.mimaTable = append(db.mimaTable, mima)
		db.mimaTable = append(db.mimaTable[:i], db.mimaTable[i+1:]...)
	}
	if needWriteFrag {
		err = db.sealAndWriteFrag(mima, Update)
	}
	return
}

// TrashByID 软删除一个 mima, 并生成一块数据库碎片.
func (db *DB) TrashByID(id string) error {
	_, mima, err := db.GetByID(id)
	if err != nil {
		return err
	}
	mima.Delete()
	return db.sealAndWriteFrag(mima, SoftDelete)
}

// UnDeleteByID 从回收站中还原一个 mima (DeletedAt 重置为零), 并生成一块数据库碎片.
// 此时, 需要判断 Alias 有无冲突, 如有冲突则清空本条记录的 Alias.
func (db *DB) UnDeleteByID(id string) (err error) {
	_, mima, err := db.GetByID(id)
	if err != nil {
		return err
	}
	if db.IsAliasConflicts(mima.Alias, id) {
		err = fmt.Errorf("%w: %s, 因此该记录的 alias 已被清空", ErrAliasConflicts, mima.Alias)
		mima.Alias = ""
	}
	mima.UnDelete()
	if err2 := db.sealAndWriteFrag(mima, UnDelete); err2 != nil {
		return err2
	}
	return
}

// IsAliasConflicts 判断 alias 有无冲突.
// 由于在设计上规定新增记录时不可编辑 alias, 只有在 edit 页面才能编辑 alias,
// 因此必然有一个本记录的 ID, 在判断冲突时应被排除在外.
func (db *DB) IsAliasConflicts(alias string, excludedID string) (yes bool) {
	if m := db.getByAlias(alias); m != nil && m.ID != excludedID {
		return true // Yes, conflicts.
	}
	return false // No, does not conflict.
}

func (db *DB) getByAlias(alias string) *Mima {
	if alias == "" {
		return nil
	}
	for i := 1; i < db.Len(); i++ {
		mima := db.mimaTable[i]
		if mima.IsDeleted() {
			continue
		}
		if mima.Alias == alias {
			return mima
		}
	}
	return nil
}

func (db *DB) sealAndWriteFrag(mima *Mima, op Operation) error {
	mima.Operation = op
	sealed, err := mima.Seal(db.key)
	if err != nil {
		return err
	}
	return db.writeFragFile(sealed)
}

// 把已加密的数据写到一个新文件中 (即生成一个新的数据库碎片).
func (db *DB) writeFragFile(box64 string) error {
	fragmentPath := filepath.Join(db.BackupDir, newTimestampFilename(FragExt))
	return writeFile(fragmentPath, box64)
}

// DeleteForeverByID 彻底删除一条记录, 并生成一块数据库碎片.
func (db *DB) DeleteForeverByID(id string) error {
	mima, err := db.deleteByID(id)
	if err != nil {
		return err
	}
	return db.sealAndWriteFrag(mima, DeleteForever)
}

func (db *DB) DeleteHistoryItem(id string, datetime string) error {
	_, mima, err := db.GetByID(id)
	if err != nil {
		return err
	}
	if err = mima.DeleteHistory(datetime); err != nil {
		return err
	}
	return db.sealAndWriteFrag(mima, Update)
}
