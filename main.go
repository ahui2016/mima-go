package main

import (
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

var (
	createAccount = "/api/create-account/"
)

func main() {
	http.HandleFunc(createAccount, createHandler)

	fmt.Println(listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func withPattern(pattern string, fn func(httpRW, httpReq, string)) httpHF {
	return func(w httpRW, r httpReq) {
		fn(w, r, pattern)
	}
}

func createHandler(w httpRW, r httpReq) {
	if r.Method != http.MethodPost {
		checkTmpl(w, templates.ExecuteTemplate(w, "create-account", nil))
		return
	}
	password := r.FormValue("password")
	if password == "" {
		form := &NormalForm{Err: errors.New("密码不能为空")}
		checkTmpl(w, templates.ExecuteTemplate(w, "create-account", form))
		return
	}
}

func checkTmpl(w httpRW, err error) {
	if err != nil {
		fmt.Fprintf(w, "%v", err)
	}
}
