package main

// Feedback 用来表示一个普通的表单.
type Feedback struct {
	Msg  string
	Err  error
	Info error
}

// MimaForm 用来表示一个 Mima, 但只包含一部分信息.
type MimaForm struct {
	ID        string
	Title     string
	Alias     string
	Username  string
	Password  string
	Notes     string
	CreatedAt string
	UpdatedAt string
	DeletedAt string
	History   []*History
	Err       error
	Info      error
}

// HidePasswordNotes 删除密码和备注, 用于不需要展示密码的页面 (为了提高安全性).
func (form *MimaForm) HidePasswordNotes() *MimaForm {
	if len(form.Password) > 0 {
		form.Password = "******"
	}
	form.Notes = ""
	return form
}

// IsDeleted 检查该 form 所对应的 mima 是否已被软删除.
func (form *MimaForm) IsDeleted() bool {
	return form.DeletedAt != ""
}
