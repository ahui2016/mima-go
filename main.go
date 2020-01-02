package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"net/http"
)

type (
	httpRW  = http.ResponseWriter
	httpReq = *http.Request
	httpHF  = http.HandlerFunc
)

func main() {
	http.HandleFunc("/create-account", createAccount)
	http.HandleFunc("/login", loginHandler)

	fmt.Println(listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func withPattern(pattern string, fn func(httpRW, httpReq, string)) httpHF {
	return func(w httpRW, r httpReq) {
		fn(w, r, pattern)
	}
}

func createAccount(w httpRW, r httpReq) {
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
	msg := &Feedback{Msg: "成功创建新账号, 请登录"}
	checkErr(w, templates.ExecuteTemplate(w, "login", msg))
}

func loginHandler(w httpRW, r httpReq) {
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "login", nil))
		return
	}
	if !isLoggedOut() {
		err := &Feedback{Err: errors.New("已登入, 不可重复登入")}
		checkErr(w, templates.ExecuteTemplate(w, "login", err))
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
	fmt.Fprintf(w, "%s", db.GetByID(0).Notes)
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
