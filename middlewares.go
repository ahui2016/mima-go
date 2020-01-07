package main

import (
	"errors"
	"time"
)

func withPattern(pattern string, fn func(httpRW, httpReq, string)) httpHF {
	return func(w httpRW, r httpReq) {
		fn(w, r, pattern)
	}
}

func dbLock(fn httpHF) httpHF {
	db.Lock()
	defer db.Unlock()
	return func(w httpRW, r httpReq) {
		fn(w, r)
	}
}

func dbRLock(fn httpHF) httpHF {
	db.RLock()
	defer db.RUnlock()
	return func(w httpRW, r httpReq) {
		fn(w, r)
	}
}

func checkState(fn httpHF) httpHF {
	return func(w httpRW, r httpReq) {
		if !isLoggedOut() {
			expired := db.StartedAt.Add(db.Period)
			if time.Now().After(expired) {
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

		if dbFileIsNotExist() {
			// 数据库不存在, 需要创建新账号.
			checkErr(w, templates.ExecuteTemplate(w, "create-account", nil))
		} else {
			// 已存在数据库, 但未登入(已登出)
			checkErr(w, templates.ExecuteTemplate(w, "login", nil))
		}
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
