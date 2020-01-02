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
	Notes     string
	Favorite  bool
	CreatedAt string
	UpdatedAt string
}

// DeletedMimas 用来表示一个已删除的 Mima, 但只包含一部分信息.
type DeletedMimas struct {
	ID        int
	Title     string
	Alias     string
	Username  string
	Notes     string
	DeletedAt string
}
