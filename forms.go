package main

import (
	"encoding/base64"
	"encoding/json"
)

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

// Settings 用来表示程序的设定, 暂时主要用于云备份.
type Settings struct {
	ApiKey            string
	ServiceInstanceID string // resource_instance_id
	AuthEndpoint      string
	ServiceEndpoint   string
	BucketLocation    string
	BucketName        string

	// Object keys can be up to 1024 characters in length, and it's best to avoid
	// any characters that might be problematic in a web address. For example, ?, =, <,
	// and other special characters might cause unwanted behavior if not URL-encoded.
	ObjKeyPrefix string // 用半角括号括住, 详见 COS.makeObjKey

	ErrMsg string
}

// CloudInfo 用来表示云端文件的信息.
type CloudInfo struct {
	CloudServiceName string
	BucketName       string
	ObjectName       string
	LastModified     string
	Err              string
	Info             string
}

func NewSettingsFromJSON64(settings64 string) (*Settings, error) {
	settingsJSON, err := base64.StdEncoding.DecodeString(settings64)
	if err != nil {
		return nil, err
	}
	settings := new(Settings)
	err = json.Unmarshal(settingsJSON, settings)
	return settings, err
}
