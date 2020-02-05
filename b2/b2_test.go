package b2

import (
	"errors"
	"testing"
)

var bucket = NewBucket(
	"9b24c95abab4018a7608041d",
	"my-test-folder",
	"000b49aa41a684d0000000005",
	"K000m0AMjjIam+cYXU6oZYw3LOpI9eI",
)

// go test -v -run TestAuthorizeAccount
func TestAuthorizeAccount(t *testing.T) {
	t.Skip("一般情况下 TestGetUploadUrl 已包含对 AuthorizeAccount 的测试")
	err := bucket.AuthorizeAccount()
	var respErr ResponseError
	if errors.As(err, &respErr) {
		t.Fatal("ResponseError:", respErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Log(bucket)
}

func TestGetUploadUrl(t *testing.T) {
	t.Skip("一般情况下 TestUploadFile 已包含对 GetUploadUrl 的测试")
	if err := bucket.AuthorizeAccount(); err != nil {
		t.Fatal(err)
	}
	uploadUrl, err := bucket.GetUploadUrl()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(*uploadUrl)
}

func TestUploadFile(t *testing.T) {
	if err := bucket.AuthorizeAccount(); err != nil {
		t.Fatal(err)
	}
	uploadUrl, err := bucket.GetUploadUrl()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := uploadUrl.UploadFile("b2.go", bucket.Folder())
	if err != nil {
		t.Fatal(err)
	}
	t.Log(*resp)
}
