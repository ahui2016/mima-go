package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
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
	http.HandleFunc("/index/", noCache(checkState(indexHandler)))
	http.HandleFunc("/add/", noCache(checkState(addHandler)))
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
	db = NewMimaDB(&key)
	db.Lock()
	defer db.Unlock()
	if err := db.MakeFirstMima(); err != nil {
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
	db = NewMimaDB(&key)
	db.Lock()
	defer db.Unlock()
	if _, err := db.Rebuild(); err != nil {
		logout()
		checkErr(w, templates.ExecuteTemplate(w, "login", &Feedback{Err: err}))
		return
	}
	http.Redirect(w, r, "/index/", http.StatusFound)
}

func logoutHandler(w httpRW, r httpReq) {
	logout()
	msg := &Feedback{Msg: "已登出, 请重新登入"}
	checkErr(w, templates.ExecuteTemplate(w, "login", msg))
}

func indexHandler(w httpRW, r httpReq) {
	checkErr(w, templates.ExecuteTemplate(w, "index", db.All()))
}

func addHandler(w httpRW, r httpReq) {
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "add", nil))
		return
	}
	form := MimaForm{
		Title:    strings.TrimSpace(r.FormValue("Title")),
		Username: strings.TrimSpace(r.FormValue("Username")),
		Password: r.FormValue("Password"),
		Notes:    strings.TrimSpace(r.FormValue("Notes")),
	}
	if form.Title == "" {
		form.Err = errors.New("标题不可为空, 请填写标题")
		checkErr(w, templates.ExecuteTemplate(w, "add", form))
		return
	}
	mima, err := form.ToMima()
	if err != nil {
		form.Err = err
		checkErr(w, templates.ExecuteTemplate(w, "add", form))
		return
	}
	db.Add(mima)
	http.Redirect(w, r, "/index/", http.StatusFound)
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
		fmt.Fprintf(w, "%v", err)
	}
}

func logout() {
	db = nil
}

func isLoggedOut() bool {
	return db == nil
}
