package main

import (
	"errors"
	"time"
)

func checkState(fn httpHF) httpHF {
	return func(w httpRW, r httpReq) {
		if !isLoggedOut() {
			expired := mdb.StartedAt.Add(mdb.Period)
			if time.Now().After(expired) {
				// 已登入, 但超时.
				logout()
				err := &Feedback{Err: errors.New("超时自动登出, 请重新登录")}
				checkErr(w, templates.ExecuteTemplate(w, "login", err))
				return
			}

			// 已登入, 未超时, 重新计时.
			mdb.Lock()
			defer mdb.Unlock()
			mdb.StartedAt = time.Now()
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
