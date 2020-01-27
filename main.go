package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	mimaDB "github.com/ahui2016/mima-go/db"
	"log"
	"net/http"
	"strings"
)

type (
	httpRW  = http.ResponseWriter
	httpReq = *http.Request
	httpHF  = http.HandlerFunc
)

func main() {
	http.HandleFunc("/create-account", noCache(createAccount))
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

	fmt.Println(listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func createAccount(w httpRW, r httpReq) {
	if !isLoggedOut() || !dbFileIsNotExist() {
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
	msg := &Feedback{Msg: "成功创建新账号, 请登入"}
	checkErr(w, templates.ExecuteTemplate(w, "login", msg))
}

func loginHandler(w httpRW, r httpReq) {
	if dbFileIsNotExist() {
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
	http.Redirect(w, r, "/home/", http.StatusFound)
}

func logoutHandler(w httpRW, _ httpReq) {
	logout()
	msg := &Feedback{Msg: "已登出, 请重新登入"}
	checkErr(w, templates.ExecuteTemplate(w, "login", msg))
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
		return
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
