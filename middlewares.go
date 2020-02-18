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
		// 数据库不存在, 需要创建新账号.
		if db.FileNotExist() {
			checkErr(w, templates.ExecuteTemplate(w, "create-account", nil))
			return
		}
		// 已存在数据库, 但数据库未初始化 (未登入/已登出)
		if db.IsNotInit() {
			checkErr(w, templates.ExecuteTemplate(w, "login", nil))
			return
		}
		// 数据库超时, 或者 session 验证失败
		if db.IsExpired() || !sessionManager.Check(r) {
			// 假设客户端 A 登入成功后, 客户端 B 尝试登陆, 会导致 logout (即 A 也被强行登出).
			// 但如果 B 紧接着输入了正确密码成功登入, 则 A 也会自动再次变成已登入状态.
			// 这可以说是一个 bug, 但恰好可以发现有人尝试登入, 所以也可以说这是一个 feature.
			logout(w)
			err := &Feedback{Err: errors.New("超时或session验证失败, 请重新登录")}
			checkErr(w, templates.ExecuteTemplate(w, "login", err))
			return
		}
		// 已创建数据库, 已登入, 数据库未超时, 并且 session 验证也成功
		db.StartedAt = time.Now()
		fn(w, r)
	}
}

// checkLogin 用于 Add 和 Edit 页面, 不检查超时.
// 不检查超时, 但还是要顺便更新有效时长.
func checkLogin(fn httpHF) httpHF {
	db.Lock()
	defer db.Unlock()
	return func(w httpRW, r httpReq) {
		// 数据库不存在, 需要创建新账号.
		if db.FileNotExist() {
			checkErr(w, templates.ExecuteTemplate(w, "create-account", nil))
			return
		}
		// 未登入(已登出)
		if isLoggedOut(r) {
			checkErr(w, templates.ExecuteTemplate(w, "login", nil))
		}
		// 已登入
		db.StartedAt = time.Now()
		fn(w, r)
		return
	}
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
		if !isLoggedOut(r) && db.IsExpired() {
			// 已登入, 但超时.
			logout(w)
			http.Error(w, "超时自动登出, 请重新登录", http.StatusNotAcceptable)
			return
		}
		if isLoggedOut(r) {
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
