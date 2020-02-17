package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type SearchResult struct {
	SearchText string
	Forms      []*MimaForm
	Info       error
	Err        error
}

// Feedback 用来表示一个普通的表单.
type Feedback struct {
	Number int
	Msg    string
	Err    error
	Info   error
}

// Settings 用来表示程序的设定, 暂时主要用于云备份.
type Settings struct {
	ApiKey            string
	ServiceInstanceID string // resource_instance_id
	AuthEndpoint      string
	ServiceEndpoint   string
	BucketLocation    string
	BucketName        string

	// Object keys can be up to 1024 characters in length, and it's best to avoid
	// any characters that might be problematic in a web address. For example, ?, =, <,
	// and other special characters might cause unwanted behavior if not URL-encoded.
	ObjKeyPrefix string // 用半角括号括住, 详见 COS.makeObjKey

	// 用于从云端恢复数据到本地, 由用户指定 Object Name.
	ObjectName string

	ErrMsg string
}

// Encode to JSON, and encode to base64.
func (settings *Settings) Encode() (string, error) {
	settingsJson, err := json.Marshal(settings)
	if err != nil {
		return "", err
	}
	settings64 := base64.StdEncoding.EncodeToString(settingsJson)
	return settings64, nil
}

// CloudInfo 用来表示云端文件的信息.
type CloudInfo struct {
	CloudServiceName string
	BucketName       string
	ObjectName       string
	LastModified     string
	Err              string
	Info             string
}

func NewSettingsFromJSON64(settings64 string) (*Settings, error) {
	settingsJSON, err := base64.StdEncoding.DecodeString(settings64)
	if err != nil {
		return nil, err
	}
	settings := new(Settings)
	err = json.Unmarshal(settingsJSON, settings)
	return settings, err
}

type SessionManager struct {
	sync.Mutex
	store     map[string]bool
	name      string
	validTerm time.Duration // 有效时长
}

// NewSessionManager 简单地返回一个的 session manager, 其中 name 固定为 "SID".
func NewSessionManager(validTerm time.Duration) *SessionManager {
	return &SessionManager{
		store:     make(map[string]bool),
		name:      "SID",
		validTerm: validTerm,
	}
}

func (manager *SessionManager) NewSession(sid string) http.Cookie {
	return http.Cookie{
		Name:     manager.name,
		Value:    sid,
		Expires:  time.Now().Add(manager.validTerm),
		HttpOnly: true,
	}
}

// Add 新增一个 sid 到 store 里, 并且写入到 w 中.
// sid 应该是一个随机数, 因此本函数内不检查冲突.
func (manager *SessionManager) Add(w httpRW, sid string) {
	manager.Lock()
	defer manager.Unlock()
	session := manager.NewSession(sid)
	http.SetCookie(w, &session)
	manager.store[sid] = true
}

// Check 检查一个 request 中是否包含有效的 sid.
// 若包含有效的 sid 则返回 true.
func (manager *SessionManager) Check(r httpReq) bool {
	manager.Lock()
	defer manager.Unlock()
	cookie, err := r.Cookie(manager.name)
	if err != nil || cookie.Value == "" || !manager.store[cookie.Value] {
		return false
	}
	return true
}

// DeleteSID 删除客户端的 SID.
func (manager *SessionManager) DeleteSID(w httpRW) {
	manager.Lock()
	defer manager.Unlock()
	session := http.Cookie{
		Name:     manager.name,
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, &session)
}
