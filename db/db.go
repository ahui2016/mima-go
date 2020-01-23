// 一个带有加密功能的简陋数据库.
package db

import (
	"encoding/base64"
	"errors"
	"os"
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
// 其中 mimaTable 相当于一个数据表, interface:Mima 相当于这个数据表的 schema.
type DB struct {
	// 每次使用 DB 时注意需要上锁.
	sync.RWMutex

	// 原始数据, 按 UpdatedAt 排序, 最新(最近)的在后面.
	mimaTable []Mima

	// 由用户密码生成 userKey, 用来加密解密 key, 再用 key 去实际加密数据.
	userKey *SecretKey
	key     *SecretKey

	// 本数据库具有定时关闭功能, 这是数据库启动时刻和有效时长.
	startedAt time.Time
	validTerm time.Duration

	// 数据库文件的绝对路径, 备份文件夹的绝对路径.
	// 另外, 数据库碎片文件的后缀名和数据库备份文件的后缀名在 db/init.go 中定义.
	fullPath  string
	backupDir string
}

// NewDB 生成一个新的 DB. 此时, 内存数据库里没有数据, 也没有 key.
// 要么通过 DB.Init 生成新的数据库, 要么通过 DB.Rebuild 从文件中恢复数据库.
func NewDB(userKey *SecretKey, fullPath, backupDir string) *DB {
	if userKey == nil {
		// 因为改错误属于 "编程时" 错误, 不是 "运行时" 错误, 可在运行前处理,
		// 因此不返回错误信息, 而是让程序直接崩溃.
		panic("缺少key, 需要key")
	}
	return &DB{
		userKey:   userKey,
		startedAt: time.Now(),
		validTerm: time.Minute * 5,
		fullPath:  fullPath,
		backupDir: backupDir,
	}
}

// Init 生成第一条记录, 用于保存密码.
// 第一条记录的 ID 特殊处理, 手动设置为空字符串.
// 同时会生成数据库文件 DB.fullPath
func (db *DB) Init(mima Mima) error {
	if !db.fileNotExist() {
		return errors.New("数据库文件已存在, 不可重复创建")
	}
	key := newRandomKey()
	db.key = &key
	mima.SetID("")
	mima.SetPassword(base64.StdEncoding.EncodeToString(key[:]))
	mima.SetNotes(randomString())
	db.mimaTable = []Mima{mima}
	box64bytes, err := mima.Seal(db.userKey) // 第一条记录特殊处理, 用 userKey 加密.
	if err != nil {
		return err
	}
	return writeFile(db.fullPath, box64bytes)
}

func (db *DB) fileNotExist() bool {
	_, err := os.Stat(db.fullPath)
	if os.IsNotExist(err) {
		return true
	}
	if err != nil {
		panic(err)
	}
	return false
}
