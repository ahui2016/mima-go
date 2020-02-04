package b2

import (
	"errors"
	"testing"
)

var bucket = NewBucket(
	"9b24c95abab4018a7608041d",
	"000b49aa41a684d0000000005",
	"K000m0AMjjIam+cYXU6oZYw3LOpI9eI",
)

// go test -v -run TestAuthorizeAccount
func TestAuthorizeAccount(t *testing.T) {
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
	if err := bucket.AuthorizeAccount(); err != nil {
		t.Fatal(err)
	}
	uploadUrl, err := bucket.GetUploadUrl()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(*uploadUrl)
}
