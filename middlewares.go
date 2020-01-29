package main

import (
	"errors"
	"net/http"
	"time"
)

func checkState(fn httpHF) httpHF {
	db.Lock()
	defer db.Unlock()
	return func(w httpRW, r httpReq) {
		if !isLoggedOut() {
			if isExpired() {
				// 已登入, 但超时.
				logout()
				err := &Feedback{Err: errors.New("超时自动登出, 请重新登录")}
				checkErr(w, templates.ExecuteTemplate(w, "login", err))
				return
			}

			// 已登入, 未超时, 重新计时.
			db.StartedAt = time.Now()
			fn(w, r)
			return
		}

		if db.FileNotExist() {
			// 数据库不存在, 需要创建新账号.
			checkErr(w, templates.ExecuteTemplate(w, "create-account", nil))
		} else {
			// 已存在数据库, 但未登入(已登出)
			checkErr(w, templates.ExecuteTemplate(w, "login", nil))
		}
	}
}

func isExpired() bool {
	db.RLock()
	defer db.RUnlock()
	expired := db.StartedAt.Add(db.ValidTerm)
	return time.Now().After(expired)
}


func noCache(fn httpHF) httpHF {
	return func(w httpRW, r httpReq) {
		w.Header().Set(
			"Cache-Control",
			"no-store, no-cache, must-revalidate",
		)
		fn(w, r)
	}
}

func copyInBackground(fn func(*Mima)) httpHF {
	db.Lock()
	defer db.Unlock()
	return func(w httpRW, r httpReq) {
		if !isLoggedOut() && isExpired() {
			// 已登入, 但超时.
			logout()
			http.Error(w, "超时自动登出, 请重新登录", http.StatusNotAcceptable)
			return
		}
		if isLoggedOut() {
			http.Error(w, "未登入(或已登出)", http.StatusNotAcceptable)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "不接受 GET 请求, 只接受 POST 请求.", http.StatusNotAcceptable)
			return
		}
		id := r.FormValue("id")
		_, mima, err := db.GetByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		db.StartedAt = time.Now()
		//noinspection GoUnhandledErrorResult
		go fn(mima)
		//if err := copyToClipboard(mima.Password); err != nil {
		//	http.Error(w, err.Error(), http.StatusInternalServerError)
		//}
	}
}
