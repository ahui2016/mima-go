package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	mimaDB "github.com/ahui2016/mima-go/db"
	"github.com/atotto/clipboard"
	"log"
	"net/http"
	"strings"
	"time"
)

type (
	httpRW  = http.ResponseWriter
	httpReq = *http.Request
	httpHF  = http.HandlerFunc
)

func main() {
	// 有 checkState 中间件的, 在 checkState 里对数据库加锁;
	// 没有 checkState 的, 要注意各自加锁.
	http.HandleFunc("/create-account", noCache(createAccount))
	http.HandleFunc("/change-password/", noCache(changePassword))
	http.HandleFunc("/login", noCache(loginHandler))
	http.HandleFunc("/logout", noCache(logoutHandler))
	http.HandleFunc("/home/", homeHandler)
	http.HandleFunc("/index/", noCache(checkState(indexHandler)))
	http.HandleFunc("/search/", noCache(checkState(searchHandler)))
	http.HandleFunc("/add/", noCache(checkState(addHandler)))
	http.HandleFunc("/delete/", noCache(checkState(deleteHandler)))
	http.HandleFunc("/recyclebin/", noCache(checkState(recyclebin)))
	http.HandleFunc("/undelete/", noCache(checkState(undeleteHandler)))
	http.HandleFunc("/delete-forever/", noCache(checkState(deleteForever)))
	http.HandleFunc("/edit/", noCache(checkState(editHandler)))
	http.HandleFunc("/api/new-password", newPassword)
	http.HandleFunc("/api/delete-history", checkState(deleteHistory))
	http.HandleFunc("/api/copy-password", copyInBackground(copyPassword))
	http.HandleFunc("/api/copy-username", copyInBackground(copyUsername))

	fmt.Println(listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func createAccount(w httpRW, r httpReq) {
	if !isLoggedOut() || !db.FileNotExist() {
		err := &Feedback{Err: errors.New("已存在账号, 不可重复创建")}
		checkErr(w, templates.ExecuteTemplate(w, "create-account", err))
		return
	}
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "create-account", nil))
		return
	}
	password := r.FormValue("password")
	if password == "" {
		err := &Feedback{Err: errors.New("密码不能为空")}
		checkErr(w, templates.ExecuteTemplate(w, "create-account", err))
		return
	}
	key := sha256.Sum256([]byte(password))
	if err := db.Init(&key); err != nil {
		checkErr(w, templates.ExecuteTemplate(w, "create-account", &Feedback{Err: err}))
		return
	}
	logout()
	info := &Feedback{Info: errors.New("成功创建新账号, 请登入")}
	checkErr(w, templates.ExecuteTemplate(w, "login", info))
}

func changePassword(w httpRW, r httpReq) {
	if isLoggedOut() || db.FileNotExist() {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "change-password", nil))
		return
	}
	oldPwd := r.FormValue("old-pwd")
	key := sha256.Sum256([]byte(oldPwd))
	if !db.EqualToUserKey(key) {
		err := &Feedback{Err: errors.New("当前密码错误, 为了提高安全性必须输入正确的当前密码")}
		checkErr(w, templates.ExecuteTemplate(w, "change-password", err))
		return
	}
	newPwd := r.FormValue("new-pwd")
	if err := db.ChangeUserKey(newPwd); err != nil {
		checkErr(w, templates.ExecuteTemplate(w, "change-password", &Feedback{Err: err}))
		return
	}
	info := &Feedback{Info: errors.New("密码修改成功, 请使用新密码登入")}
	checkErr(w, templates.ExecuteTemplate(w, "login", info))
}

func loginHandler(w httpRW, r httpReq) {
	db.Lock()
	defer db.Unlock()
	if db.FileNotExist() {
		// 数据库不存在, 需要创建新账号.
		checkErr(w, templates.ExecuteTemplate(w, "create-account", nil))
		return
	}
	if !isLoggedOut() {
		err := &Feedback{Err: errors.New("已登入, 不可重复登入")}
		checkErr(w, templates.ExecuteTemplate(w, "login", err))
		return
	}
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "login", nil))
		return
	}
	password := r.FormValue("password")
	key := sha256.Sum256([]byte(password))
	db.Reset()
	if _, err := db.Rebuild(&key); err != nil {
		logout()
		checkErr(w, templates.ExecuteTemplate(w, "login", &Feedback{Err: err}))
		return
	}

	// 必须更新时间, 这是容易忽略出错的地方.
	// 如果不更新时间, 会出现 "未登入, 已超时" 的错误.
	db.StartedAt = time.Now()

	http.Redirect(w, r, "/home/", http.StatusFound)
}

func logoutHandler(w httpRW, _ httpReq) {
	logout()
	info := &Feedback{Info: errors.New("已登出, 请重新登入")}
	checkErr(w, templates.ExecuteTemplate(w, "login", info))
}

func homeHandler(w httpRW, r httpReq) {
	http.Redirect(w, r, "/index/", http.StatusFound)
}

func indexHandler(w httpRW, _ httpReq) {
	checkErr(w, templates.ExecuteTemplate(w, "index", db.All()))
}

func searchHandler(w httpRW, r httpReq) {
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "search", nil))
		return
	}
	alias := strings.TrimSpace(r.FormValue("alias"))
	form := new(MimaForm)
	if alias == "" {
		form.Info = errors.New(
			"不可搜索空字符串, 请输入完整的别名, 本程序只能精确搜索, 区分大小写")
		result := &SearchResult{MimaForm: form}
		checkErr(w, templates.ExecuteTemplate(w, "search", result))
		return
	}
	form = db.GetFormByAlias(alias)
	result := &SearchResult{SearchText: alias, MimaForm: form}
	checkErr(w, templates.ExecuteTemplate(w, "search", result))
}

func recyclebin(w httpRW, _ httpReq) {
	checkErr(w, templates.ExecuteTemplate(w, "recyclebin", db.DeletedMimas()))
}

func addHandler(w httpRW, r httpReq) {
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "add", nil))
		return
	}
	form := &MimaForm{
		Title:    strings.TrimSpace(r.FormValue("Title")),
		Username: strings.TrimSpace(r.FormValue("Username")),
		Password: r.FormValue("Password"),
		Notes:    strings.TrimSpace(r.FormValue("Notes")),
	}
	mima, err := mimaDB.NewMimaFromForm(form)
	if err == nil {
		err = db.Add(mima)
	}
	if err != nil {
		form.Err = err
		checkErr(w, templates.ExecuteTemplate(w, "add", form))
		return
	}
	http.Redirect(w, r, "/home/", http.StatusFound)
}

func editHandler(w httpRW, r httpReq) {
	form := new(MimaForm)
	id, ok := getAndCheckID(w, r, "edit", form)
	if !ok {
		return
	}
	form = db.GetFormByID(id)
	if r.Method != http.MethodPost {
		if form.IsDeleted() {
			form = &MimaForm{Err: errMimaDeleted}
		}
		checkErr(w, templates.ExecuteTemplate(w, "edit", form))
		return
	}
	form = &MimaForm{
		ID:       id,
		Title:    strings.TrimSpace(r.FormValue("Title")),
		Alias:    strings.TrimSpace(r.FormValue("Alias")),
		Username: strings.TrimSpace(r.FormValue("Username")),
		Password: r.FormValue("Password"),
		Notes:    strings.TrimSpace(r.FormValue("Notes")),
		History:  form.History,
	}
	if form.Err = db.Update(form); form.Err != nil {
		checkErr(w, templates.ExecuteTemplate(w, "edit", form))
		return
	}
	result := &SearchResult{MimaForm: form}
	checkErr(w, templates.ExecuteTemplate(w, "search", result))
}

func getAndCheckID(w httpRW, r httpReq, tmpl string, form *MimaForm) (id string, ok bool) {
	if id = strings.TrimSpace(r.FormValue("id")); id == "" {
		form.Err = fmt.Errorf("id 不可为空")
		checkErr(w, templates.ExecuteTemplate(w, tmpl, form))
		return
	}
	return id, true
}

func deleteHandler(w httpRW, r httpReq) {
	form := new(MimaForm)
	id, ok := getAndCheckID(w, r, "delete", form)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		form = db.GetFormByID(id).HideSecrets()
		if form.IsDeleted() {
			form = &MimaForm{Err: errMimaDeleted}
		}
		checkErr(w, templates.ExecuteTemplate(w, "delete", form))
		return
	}
	if err := db.TrashByID(id); err != nil {
		form.Err = err
		checkErr(w, templates.ExecuteTemplate(w, "delete", form))
		return
	}
	http.Redirect(w, r, "/home/", http.StatusFound)
}

func undeleteHandler(w httpRW, r httpReq) {
	form := new(MimaForm)
	id, ok := getAndCheckID(w, r, "undelete", form)
	if !ok {
		return
	}
	form = db.GetFormByID(id)
	if !form.IsDeleted() {
		form := &MimaForm{Err: errors.New("回收站中找不到此记录: " + id)}
		checkErr(w, templates.ExecuteTemplate(w, "undelete", form))
		return
	}
	if r.Method != http.MethodPost {
		if db.IsAliasConflicts(form.Alias, id) {
			form.Info = fmt.Errorf(
				"%w: %s, 如果确认还原此记录, 该 alias 将被清空",
				mimaDB.ErrAliasConflicts, form.Alias)
		}
		checkErr(w, templates.ExecuteTemplate(w, "undelete", form))
		return
	}
	err := db.UnDeleteByID(id)
	if err != nil && !errors.Is(err, mimaDB.ErrAliasConflicts) {
		form = &MimaForm{Err: err}
		checkErr(w, templates.ExecuteTemplate(w, "undelete", form))
		return
	}
	if errors.Is(err, mimaDB.ErrAliasConflicts) {
		form.Info = err
		form.Alias = ""
	}
	checkErr(w, templates.ExecuteTemplate(w, "edit", form))
}

func deleteForever(w httpRW, r httpReq) {
	form := new(MimaForm)
	id, ok := getAndCheckID(w, r, "delete-forever", form)
	if !ok {
		return
	}
	form = db.GetFormByID(id)
	if !form.IsDeleted() {
		form := &MimaForm{Err: errors.New("回收站中找不到此记录: " + id)}
		checkErr(w, templates.ExecuteTemplate(w, "delete-forever", form))
		return
	}
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "delete-forever", form))
		return
	}
	checkErr(w, db.DeleteForeverByID(id))
	http.Redirect(w, r, "/recyclebin/", http.StatusFound)
}

func deleteHistory(w httpRW, r httpReq) {
	id := strings.TrimSpace(r.FormValue("id"))
	if id == "" {
		http.Error(w, "id 不可为空", http.StatusNotAcceptable)
		return
	}
	datetime := strings.TrimSpace(r.FormValue("datetime"))
	if len(datetime) < len(mimaDB.DateTimeFormat) {
		http.Error(w, fmt.Sprintf("格式错误: %s", datetime), http.StatusConflict)
		return
	}
	if err := db.DeleteHistoryItem(id, datetime); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
	}
}

/*
func writeJSON(w httpRW, msg string, status int) {
	w.WriteHeader(status)
	checkErr(w, json.NewEncoder(w).Encode(msg))
}
*/

func newPassword(w httpRW, _ httpReq) {
	pwBytes := make([]byte, passwordSize)
	if _, err := rand.Read(pwBytes); err != nil {
		_, _ = fmt.Fprint(w, err)
	}
	pw := base64.RawURLEncoding.EncodeToString(pwBytes)[:passwordSize]
	_, _ = fmt.Fprint(w, pw)
}

func copyPassword(mima *Mima) {
	_ = copyToClipboard(mima.Password)
}

func copyUsername(mima *Mima) {
	_ = copyToClipboard(mima.Username)
}

func checkErr(w httpRW, err error) {
	if err != nil {
		log.Println(err)
		_, _ = fmt.Fprintf(w, "%v", err)
	}
}

func logout() {
	db.Reset()
}

func isLoggedOut() bool {
	return db.IsNotInit()
}

// 复制到剪贴板, 并在一定时间后清空剪贴板.
func copyToClipboard(s string) (err error) {
	if err = clipboard.WriteAll(s); err != nil {
		return
	}

	// 三十秒后自动清空剪贴板.
	<-time.After(time.Second * 30)

	var text string
	if text, err = clipboard.ReadAll(); err != nil {
		return
	}
	if text == s {
		return clipboard.WriteAll("")
	}
	return nil
}
