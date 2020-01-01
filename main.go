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
		form := &NormalForm{Err: errors.New("密码不能为空")}
		checkErr(w, templates.ExecuteTemplate(w, "create-account", form))
		return
	}
	key := sha256.Sum256([]byte(password))
	db = NewMimaDB(&key)
	db.Lock()
	defer db.Unlock()
	if err := db.MakeFirstMima(); err != nil {
		form := &NormalForm{Err: err}
		checkErr(w, templates.ExecuteTemplate(w, "create-account", form))
	}
	fmt.Fprintln(w, "成功创建新账号")
}

func checkErr(w httpRW, err error) {
	if err != nil {
		fmt.Fprintf(w, "%v", err)
	}
}
