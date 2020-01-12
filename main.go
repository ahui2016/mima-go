package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
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
	http.HandleFunc("/add/", noCache(checkState(addHandler)))
	http.HandleFunc("/delete/", noCache(checkState(deleteHandler)))
	http.HandleFunc("/recyclebin/", noCache(checkState(recyclebin)))
	http.HandleFunc("/undelete/", noCache(checkState(undeleteHandler)))
	http.HandleFunc("/edit/", noCache(checkState(editHandler)))
	http.HandleFunc("/api/new-password", newPassword)

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
	mdb = NewMimaDB(&key)
	if err := mdb.MakeFirstMima(); err != nil {
		checkErr(w, templates.ExecuteTemplate(w, "create-account", &Feedback{Err: err}))
		return
	}
	logout()
	msg := &Feedback{Msg: "成功创建新账号, 请登入"}
	checkErr(w, templates.ExecuteTemplate(w, "login", msg))
}

func loginHandler(w httpRW, r httpReq) {
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
	mdb = NewMimaDB(&key)
	if _, err := mdb.Rebuild(); err != nil {
		logout()
		checkErr(w, templates.ExecuteTemplate(w, "login", &Feedback{Err: err}))
		return
	}
	http.Redirect(w, r, "/home/", http.StatusFound)
}

func logoutHandler(w httpRW, r httpReq) {
	logout()
	msg := &Feedback{Msg: "已登出, 请重新登入"}
	checkErr(w, templates.ExecuteTemplate(w, "login", msg))
}

func homeHandler(w httpRW, r httpReq) {
	http.Redirect(w, r, "/index/", http.StatusFound)
}

func indexHandler(w httpRW, r httpReq) {
	checkErr(w, templates.ExecuteTemplate(w, "index", mdb.All()))
}

func recyclebin(w httpRW, r httpReq) {
	checkErr(w, templates.ExecuteTemplate(w, "recyclebin", mdb.DeletedMimas()))
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
	mima, err := NewMimaFromForm(form)
	if err == nil {
		err = mdb.Add(mima)
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
	if r.Method != http.MethodPost {
		form = mdb.GetFormByID(id)
		if form.IsDeleted() {
			form = &MimaForm{Err: errors.New("此记录已被删除, 不可编辑")}
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
	}
	// if err := mdb.Update(form); err != nil {
	// 	form.Err = err
	// 	checkErr(w, templates.ExecuteTemplate(w, "edit", form))
	// 	return
	// }
}

func getAndCheckID(w httpRW, r httpReq, tmpl string, form *MimaForm) (id int, ok bool) {
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		form.Err = err
		checkErr(w, templates.ExecuteTemplate(w, tmpl, form))
		return
	}
	if id <= 0 || id >= mdb.NextID {
		form.Err = fmt.Errorf("id: %d out of range", id)
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
		form = mdb.GetFormByID(id)
		if form.IsDeleted() {
			form.Err = errors.New("此记录已被删除, 不可重复删除")
		}
		checkErr(w, templates.ExecuteTemplate(w, "delete", form))
		return
	}
	if err := mdb.TrashByID(id); err != nil {
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
	form = mdb.GetFormByID(id)
	if !form.IsDeleted() {
		form.Err = errors.New("此记录不在回收站中")
	}
	if r.Method != http.MethodPost {
		if mdb.IsAliasExist(form.Alias) {
			form.Info = fmt.Errorf(
				"%w: %s, 如果确认还原此记录, 该 alias 将被清空", errAliasExist, form.Alias)
		}
		checkErr(w, templates.ExecuteTemplate(w, "undelete", form))
		return
	}
	err := mdb.UndeleteByID(id)
	if errors.Is(err, errAliasExist) {
		form.Info = err
		checkErr(w, templates.ExecuteTemplate(w, "edit", form))
		return
	}
	if err != nil {
		form.Err = err
		checkErr(w, templates.ExecuteTemplate(w, "undelete", form))
		return
	}
}

func newPassword(w httpRW, r httpReq) {
	pwBytes := make([]byte, passwordSize)
	if _, err := rand.Read(pwBytes); err != nil {
		fmt.Fprint(w, err)
	}
	pw := base64.RawURLEncoding.EncodeToString(pwBytes)[:passwordSize]
	fmt.Fprint(w, pw)
}

func checkErr(w httpRW, err error) {
	if err != nil {
		log.Println(err)
		fmt.Fprintf(w, "%v", err)
	}
}

func logout() {
	mdb = nil
}

func isLoggedOut() bool {
	return mdb == nil
}
