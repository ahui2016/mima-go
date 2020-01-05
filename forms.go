package main

import ()

// Feedback 用来表示一个普通的表单.
type Feedback struct {
	Msg string
	Err error
}

// MimaForm 用来表示一个 Mima, 但只包含一部分信息.
type MimaForm struct {
	ID        int
	Title     string
	Alias     string
	Username  string
	Password  string
	Notes     string
	CreatedAt string
	UpdatedAt string
	DeletedAt string
	Err       error
}

// HidePasswordNotes 删除密码和备注, 用于不需要展示密码的页面 (为了提高安全性).
func (form *MimaForm) HidePasswordNotes() *MimaForm {
	if len(form.Password) > 0 {
		form.Password = "******"
	}
	form.Notes = ""
	return form
}

// ToMima 把 MimaForm 转换为 Mima.
func (form *MimaForm) ToMima() (mima *Mima, err error) {
	if mima, err = NewMima(form.Title); err != nil {
		return
	}
	mima.Username = form.Username
	mima.Password = form.Password
	mima.Notes = form.Notes
	return
}
