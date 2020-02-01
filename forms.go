package main

type SearchResult struct {
	SearchText string
	Forms      []*MimaForm
	Info       error
	Err        error
}

type AjaxResponse struct {
	Message string
}

// Feedback 用来表示一个普通的表单.
type Feedback struct {
	Number int
	Msg    string
	Err    error
	Info   error
}
